package backend

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

const leadNotFoundMessage = "Lead nicht gefunden."

// --- Auth ---

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !s.auth.configured() {
		log.Printf("login attempted but ADMIN_EMAIL / ADMIN_PASSWORD_HASH / SESSION_SECRET are not configured")
		writeError(w, http.StatusServiceUnavailable, "Der Login ist nicht konfiguriert.")
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage.")
		return
	}
	if !s.auth.checkCredentials(body.Email, body.Password) {
		log.Printf("failed admin login from %s", r.RemoteAddr)
		writeError(w, http.StatusUnauthorized, "E-Mail oder Passwort ist falsch.")
		return
	}

	s.auth.setCookie(w, s.auth.newSession(), int(sessionTTL.Seconds()))
	writeJSON(w, http.StatusOK, map[string]string{"email": s.auth.email})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.auth.setCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	email, _ := r.Context().Value(ctxKey{}).(string)
	writeJSON(w, http.StatusOK, map[string]string{"email": email})
}

// --- Leads ---

func (s *Server) handleListLeads(w http.ResponseWriter, r *http.Request) {
	f, err := parseLeadFilter(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page := atoiDefault(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := atoiDefault(r.URL.Query().Get("pageSize"), defaultPageSize)
	if pageSize < 1 || pageSize > maxPageSize {
		pageSize = defaultPageSize
	}
	f.Limit, f.Offset = pageSize, (page-1)*pageSize

	leads, total, err := s.repo.ListLeads(r.Context(), f)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"leads":    leads,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (s *Server) handleGetLead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validUUID(id) {
		writeError(w, http.StatusNotFound, leadNotFoundMessage)
		return
	}
	detail, err := s.repo.GetLead(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, leadNotFoundMessage)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validUUID(id) {
		writeError(w, http.StatusNotFound, leadNotFoundMessage)
		return
	}
	var body struct {
		Status LeadStatus `json:"status"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage.")
		return
	}
	if !body.Status.Valid() {
		writeError(w, http.StatusUnprocessableEntity, "Ungültiger Status.")
		return
	}

	err := s.repo.UpdateStatus(r.Context(), id, body.Status)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, leadNotFoundMessage)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": string(body.Status)})
}

// handleExport streams all leads matching the current dashboard filters.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	f, err := parseLeadFilter(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	leads, _, err := s.repo.ListLeads(r.Context(), f) // Limit 0 = all
	if err != nil {
		internalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+exportFilename(time.Now())+`"`)
	if err := WriteLeadsCSV(w, leads); err != nil {
		// Headers are already sent; the download will be incomplete.
		log.Printf("csv export failed: %v", err)
	}
}

func (s *Server) handleCampaignStats(w http.ResponseWriter, r *http.Request) {
	days := atoiDefault(r.URL.Query().Get("days"), 0) // 0 = all time
	stats, err := s.repo.CampaignStats(r.Context(), days)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaigns": stats})
}

func (s *Server) handleWeeklyLeads(w http.ResponseWriter, r *http.Request) {
	weeks := atoiDefault(r.URL.Query().Get("weeks"), 8)
	if weeks < 1 || weeks > 52 {
		weeks = 8
	}
	labels, rows, err := s.repo.WeeklyLeadsBySource(r.Context(), weeks)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"weeks": labels, "rows": rows})
}

func (s *Server) handleDeleteLead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validUUID(id) {
		writeError(w, http.StatusNotFound, leadNotFoundMessage)
		return
	}
	err := s.repo.DeleteLead(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, leadNotFoundMessage)
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	// Log only the ID: deleted personal data must not live on in the logs.
	email, _ := r.Context().Value(ctxKey{}).(string)
	log.Printf("lead %s deleted by %s", id, email)
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

type filterError string

func (e filterError) Error() string { return string(e) }

func parseLeadFilter(q url.Values) (LeadFilter, error) {
	f := LeadFilter{
		Search:         q.Get("search"),
		Source:         q.Get("source"),
		Campaign:       q.Get("campaign"),
		HideDuplicates: q.Get("hideDuplicates") == "true",
		Sort:           q.Get("sort"),
		Desc:           q.Get("order") != "asc",
	}
	if f.Sort == "" {
		f.Sort = "created_at"
	}
	if _, ok := sortColumns[f.Sort]; !ok {
		return f, filterError("Ungültige Sortierung.")
	}
	if st := q.Get("status"); st != "" {
		f.Status = LeadStatus(st)
		if !f.Status.Valid() {
			return f, filterError("Ungültiger Status.")
		}
	}
	return f, nil
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// validUUID avoids sending malformed IDs to Postgres, which would answer
// with a type error (and we'd return 500 instead of 404).
func validUUID(s string) bool { return uuidRe.MatchString(s) }

func atoiDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func internalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "Interner Fehler. Bitte später erneut versuchen.")
}
