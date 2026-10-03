package backend

import (
	"context"
	"errors"
	"log"
	"strings"
)

type leadStore interface {
	CreateLead(context.Context, Lead) (Lead, error)
}

type geoSaver interface {
	SaveGeo(ctx context.Context, id string, g GeoResult) error
}

type LeadService struct {
	repo leadStore

	// Optional geo enrichment (nil = disabled, e.g. no GEOAPIFY_API_KEY).
	geocoder Geocoder
	geoStore geoSaver
}

// WithGeocoder enables geo enrichment of new leads.
func (s *LeadService) WithGeocoder(g Geocoder, store geoSaver) *LeadService {
	s.geocoder, s.geoStore = g, store
	return s
}

func NewLeadService(repo leadStore) *LeadService {
	return &LeadService{repo: repo}
}

// Create runs the submission flow: validate → normalize → store (with
// duplicate check). Only a database failure makes the request fail.
func (s *LeadService) Create(ctx context.Context, req CreateLeadRequest) (Lead, error) {
	if verr := Validate(req); verr != nil {
		return Lead{}, verr
	}

	lead := buildLead(req)

	lead, err := s.repo.CreateLead(ctx, lead)
	if err != nil {
		return Lead{}, err
	}

	if lead.DuplicateOf != nil {
		log.Printf("lead %s stored as possible duplicate of %s", lead.ID, *lead.DuplicateOf)
	}

	// Geo enrichment runs after the lead is safely stored. It is bounded in
	// time and its outcome never changes the result of the submission. It
	// runs before the response because serverless runtimes may freeze
	// background work once the response is sent.
	if s.geocoder != nil {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), enrichTimeout)
		s.Enrich(ctx, lead.ID, Address{
			Street: lead.Street, HouseNumber: lead.HouseNumber, PostalCode: lead.PostalCode, City: lead.City,
		})
		cancel()
	}
	return lead, nil
}

// Enrich geocodes one address and stores the outcome, including failures
// (status "not_found" / "error"), so they are visible and can be retried.
func (s *LeadService) Enrich(ctx context.Context, leadID string, a Address) GeoResult {
	if s.geocoder == nil {
		return GeoResult{}
	}
	res, err := s.geocoder.Geocode(ctx, a)
	switch {
	case errors.Is(err, ErrNoGeoMatch):
		res = GeoResult{Status: GeoStatusNotFound}
	case err != nil:
		log.Printf("geocoding lead %s failed: %v", leadID, err)
		res = GeoResult{Status: GeoStatusError}
	}
	// Save with its own short deadline: when the geocoder used up ctx (e.g.
	// a timeout), the outcome must still be recorded so it can be retried.
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveGeoTimeout)
	defer cancel()
	if err := s.geoStore.SaveGeo(saveCtx, leadID, res); err != nil {
		log.Printf("saving geo data for lead %s failed: %v", leadID, err)
	}
	return res
}

// buildLead trims raw values (keeping the customer's spelling) and computes
// normalized values for duplicate detection. Expects a validated request.
func buildLead(req CreateLeadRequest) Lead {
	t := strings.TrimSpace
	houseNumber := strings.Join(strings.Fields(req.HouseNumber), " ")

	phone, phoneNormalized := t(req.Phone), NormalizePhone(req.Phone)
	if req.PhoneCountryCode != "" {
		phone, phoneNormalized = ComposePhone(req.PhoneCountryCode, req.Phone)
	}

	return Lead{
		FirstName: t(req.FirstName),
		LastName:  t(req.LastName),

		Email:           t(req.Email),
		EmailNormalized: NormalizeEmail(req.Email),

		Phone:           phone,
		PhoneNormalized: phoneNormalized,

		Street:           t(req.Street),
		StreetNormalized: NormalizeStreet(req.Street),

		HouseNumber:           houseNumber,
		HouseNumberNormalized: NormalizeHouseNumber(req.HouseNumber),

		PostalCode: NormalizePostalCode(req.PostalCode),

		City:           t(req.City),
		CityNormalized: NormalizeText(req.City),

		ParcelNote: t(req.ParcelNote),

		// Attribution must never block a submission, so overlong values are
		// truncated instead of rejected. Source, medium and campaign are
		// lowercased so "Google" and "google" end up in the same group.
		// Content and term keep their case (ad names, keywords).
		UTMSource:   strings.ToLower(truncate(req.UTMSource, 255)),
		UTMMedium:   strings.ToLower(truncate(req.UTMMedium, 255)),
		UTMCampaign: strings.ToLower(truncate(req.UTMCampaign, 255)),
		UTMContent:  truncate(req.UTMContent, 255),
		UTMTerm:     truncate(req.UTMTerm, 255),
		GCLID:       truncate(req.GCLID, 255),
		FBCLID:      truncate(req.FBCLID, 255),
		Referrer:    truncate(req.Referrer, 1000),

		Status: LeadStatusNew,
	}
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}
