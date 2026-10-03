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
// typed (trimmed); *Normalized fields are only used for duplicate detection
// and are not sent to the dashboard.
type Lead struct {
	ID string `json:"id"`

	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`

	Email           string `json:"email"`
	EmailNormalized string `json:"-"`

	Phone           string `json:"phone"`
	PhoneNormalized string `json:"-"`

	Street           string `json:"street"`
	StreetNormalized string `json:"-"`

	HouseNumber           string `json:"houseNumber"` // "" if the plot has none
	HouseNumberNormalized string `json:"-"`

	PostalCode string `json:"postalCode"`

	City           string `json:"city"`
	CityNormalized string `json:"-"`

	ParcelNote string `json:"parcelNote"` // e.g. "Flurstück 123/4"

	UTMSource   string `json:"utmSource"`
	UTMMedium   string `json:"utmMedium"`
	UTMCampaign string `json:"utmCampaign"`
	UTMContent  string `json:"utmContent"`
	UTMTerm     string `json:"utmTerm"`
	GCLID       string `json:"gclid"`
	FBCLID      string `json:"fbclid"`
	Referrer    string `json:"referrer"`

	Status      LeadStatus `json:"status"`
	DuplicateOf *string    `json:"duplicateOf"`

	CreatedAt time.Time `json:"createdAt"`
}

// LeadListItem is a lead as shown in the dashboard, plus the number of later
// leads that were marked as duplicates of it.
type LeadListItem struct {
	Lead
	DuplicateCount int     `json:"duplicateCount"`
	Geo            GeoInfo `json:"geo"`
}

// LeadSummary is a short reference to a related lead (original / duplicates).
type LeadSummary struct {
	ID        string     `json:"id"`
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	Email     string     `json:"email"`
	Status    LeadStatus `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
}

// LeadDetail is the payload of the lead detail page.
type LeadDetail struct {
	Lead       LeadListItem  `json:"lead"`
	Original   *LeadSummary  `json:"original"`   // set if this lead is a duplicate
	Duplicates []LeadSummary `json:"duplicates"` // later leads pointing to this one
}

// CampaignStats is one row of the campaign evaluation. Status counts are
// based on unique leads only, so duplicates don't inflate a campaign.
type CampaignStats struct {
	Source       string `json:"source"`
	Campaign     string `json:"campaign"`
	Total        int    `json:"total"`
	Unique       int    `json:"unique"`
	New          int    `json:"new"`
	Contacted    int    `json:"contacted"`
	Qualified    int    `json:"qualified"`
	NotQualified int    `json:"notQualified"`
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
