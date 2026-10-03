package backend

import "time"

type LeadStatus string

const (
	LeadStatusNew          LeadStatus = "new"
	LeadStatusContacted    LeadStatus = "contacted"
	LeadStatusQualified    LeadStatus = "qualified"
	LeadStatusNotQualified LeadStatus = "not_qualified"
)

func (s LeadStatus) Valid() bool {
	switch s {
	case LeadStatusNew, LeadStatusContacted, LeadStatusQualified, LeadStatusNotQualified:
		return true
	}
	return false
}

// Lead is a stored location-check request. Raw fields hold what the customer
// typed (trimmed); *Normalized fields are only used for duplicate detection.
type Lead struct {
	ID string

	FirstName string
	LastName  string

	Email           string
	EmailNormalized string

	Phone           string
	PhoneNormalized string

	Street           string
	StreetNormalized string

	HouseNumber           string // optional, "" if the plot has none
	HouseNumberNormalized string

	PostalCode string

	City           string
	CityNormalized string

	ParcelNote string // optional free text, e.g. "Flurstück 123/4"

	UTMSource   string
	UTMMedium   string
	UTMCampaign string
	UTMContent  string
	UTMTerm     string
	GCLID       string
	FBCLID      string
	Referrer    string

	Status      LeadStatus
	DuplicateOf *string

	CreatedAt time.Time
}

// CreateLeadRequest is the JSON body of POST /api/leads. Nothing in it is
// trusted: it is validated and normalized before a Lead is built from it.
type CreateLeadRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	// Optional: when set (form dropdown, e.g. "+49"), Phone is the national
	// number only. When empty, Phone must be a full number.
	PhoneCountryCode string `json:"phoneCountryCode"`

	Street      string `json:"street"`
	HouseNumber string `json:"houseNumber"`
	PostalCode  string `json:"postalCode"`
	City        string `json:"city"`
	ParcelNote  string `json:"parcelNote"`

	Consent bool `json:"consent"`

	UTMSource   string `json:"utmSource"`
	UTMMedium   string `json:"utmMedium"`
	UTMCampaign string `json:"utmCampaign"`
	UTMContent  string `json:"utmContent"`
	UTMTerm     string `json:"utmTerm"`
	GCLID       string `json:"gclid"`
	FBCLID      string `json:"fbclid"`
	Referrer    string `json:"referrer"`

	// Honeypot: hidden in the form, real users leave it empty.
	Website string `json:"website"`
}
