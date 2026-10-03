package backend

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// backfillBatch is how many leads one "geocode missing" call handles. They
// run in parallel because Geoapify needs 3–15 s per answer; the starts are
// staggered to stay below the free plan's 5 requests/second, and every
// request is bounded so the whole call ends within the 15 s server limit.
const backfillBatch = 5

// deriveGeo computes service area and data-quality signals for display.
// They are derived on read, so changing SERVICE_AREA_STATES applies to all
// existing leads immediately.
func (s *Server) deriveGeo(leads []LeadListItem) {
	for i := range leads {
		leads[i].Geo.derive(leads[i].PostalCode, s.area)
	}
}

func (s *Server) handleGeoStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.repo.GeoStats(r.Context(), s.area)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"geo": stats, "enabled": s.leads.geocoder != nil})
}

// handleGeocodeLead re-runs geocoding for one lead (e.g. after an error).
func (s *Server) handleGeocodeLead(w http.ResponseWriter, r *http.Request) {
	if s.leads.geocoder == nil {
		writeError(w, http.StatusServiceUnavailable, "Geodaten sind nicht konfiguriert (GEOAPIFY_API_KEY fehlt).")
		return
	}
	id := chi.URLParam(r, "id")
	if !validUUID(id) {
		writeError(w, http.StatusNotFound, leadNotFoundMessage)
		return
	}
	addr, err := s.repo.leadAddress(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, leadNotFoundMessage)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), adminGeocodeTimeout)
	s.leads.Enrich(ctx, id, addr)
	cancel()

	detail, err := s.repo.GetLead(r.Context(), id)
	if err != nil {
		internalError(w, err)
		return
	}
	detail.Lead.Geo.derive(detail.Lead.PostalCode, s.area)
	writeJSON(w, http.StatusOK, detail)
}

// handleGeocodeMissing geocodes leads that were never geocoded or failed
// (e.g. leads from before the feature, or while Geoapify was down).
func (s *Server) handleGeocodeMissing(w http.ResponseWriter, r *http.Request) {
	if s.leads.geocoder == nil {
		writeError(w, http.StatusServiceUnavailable, "Geodaten sind nicht konfiguriert (GEOAPIFY_API_KEY fehlt).")
		return
	}
	targets, remaining, err := s.repo.LeadsNeedingGeo(r.Context(), backfillBatch)
	if err != nil {
		internalError(w, err)
		return
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		counts = map[GeoStatus]int{}
	)
	for i, t := range targets {
		wg.Add(1)
		go func(delay time.Duration) {
			defer wg.Done()
			time.Sleep(delay)
			ctx, cancel := context.WithTimeout(r.Context(), adminGeocodeTimeout)
			defer cancel()
			res := s.leads.Enrich(ctx, t.ID, t.Address)
			mu.Lock()
			counts[res.Status]++
			mu.Unlock()
		}(time.Duration(i) * 250 * time.Millisecond)
	}
	wg.Wait()
	log.Printf("geo backfill: %d processed (%d ok, %d not found, %d errors)",
		len(targets), counts[GeoStatusOK], counts[GeoStatusNotFound], counts[GeoStatusError])

	writeJSON(w, http.StatusOK, map[string]int{
		"processed": len(targets),
		"ok":        counts[GeoStatusOK],
		"notFound":  counts[GeoStatusNotFound],
		"errors":    counts[GeoStatusError],
		// Failed ones stay in the queue, so remaining can't simply be
		// "before - processed"; report what is left to do now.
		"remaining": remaining - counts[GeoStatusOK] - counts[GeoStatusNotFound],
	})
}
