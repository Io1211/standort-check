package backend

import (
	"context"
	"log"
	"strings"
)

type LeadService struct {
	repo *Repository
}

func NewLeadService(repo *Repository) *LeadService {
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
	return lead, nil
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
		// truncated instead of rejected.
		UTMSource:   truncate(req.UTMSource, 255),
		UTMMedium:   truncate(req.UTMMedium, 255),
		UTMCampaign: truncate(req.UTMCampaign, 255),
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
