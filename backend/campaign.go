package backend

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Campaign is the configuration behind one tracking link.
type Campaign struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	UTMSource   string    `json:"utmSource"`
	UTMMedium   string    `json:"utmMedium"`
	UTMCampaign string    `json:"utmCampaign"`
	UTMContent  string    `json:"utmContent"`
	Notes       string    `json:"notes"`
	Archived    bool      `json:"archived"`
	CreatedAt   time.Time `json:"createdAt"`

	// Leads that arrived through this link (unique = without duplicates).
	Leads     int `json:"leads"`
	Unique    int `json:"unique"`
	Qualified int `json:"qualified"`
}

type CampaignRequest struct {
	Name        string `json:"name"`
	UTMSource   string `json:"utmSource"`
	UTMMedium   string `json:"utmMedium"`
	UTMCampaign string `json:"utmCampaign"`
	UTMContent  string `json:"utmContent"`
	Notes       string `json:"notes"`
	Archived    bool   `json:"archived"`
}

var (
	slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	// utm_content may also hold the ad platforms' dynamic placeholders such
	// as Meta's {{ad.name}} or Google's {creative}.
	contentRe = regexp.MustCompile(`^[A-Za-z0-9._{}-]+$`)
)

// normalize trims all values. Slugs are not silently rewritten (that would
// surprise the user); invalid ones are rejected with a clear message.
func (r CampaignRequest) normalize() CampaignRequest {
	t := strings.TrimSpace
	return CampaignRequest{
		Name:        t(r.Name),
		UTMSource:   t(r.UTMSource),
		UTMMedium:   t(r.UTMMedium),
		UTMCampaign: t(r.UTMCampaign),
		UTMContent:  t(r.UTMContent),
		Notes:       t(r.Notes),
		Archived:    r.Archived,
	}
}

func (r CampaignRequest) validate() *ValidationError {
	f := map[string]string{}
	const slugMsg = "Nur Kleinbuchstaben, Ziffern, - _ . (keine Leerzeichen oder Umlaute)."

	if r.Name == "" {
		f["name"] = "Bitte einen Namen angeben."
	} else if utf8.RuneCountInString(r.Name) > 120 {
		f["name"] = "Name ist zu lang."
	}
	for field, value := range map[string]string{
		"utmSource": r.UTMSource, "utmMedium": r.UTMMedium, "utmCampaign": r.UTMCampaign,
	} {
		switch {
		case value == "":
			f[field] = "Pflichtfeld."
		case len(value) > 100:
			f[field] = "Zu lang (max. 100 Zeichen)."
		case !slugRe.MatchString(value):
			f[field] = slugMsg
		}
	}
	if r.UTMContent != "" && (len(r.UTMContent) > 100 || !contentRe.MatchString(r.UTMContent)) {
		f["utmContent"] = "Nur Buchstaben, Ziffern, - _ . und Platzhalter wie {{ad.name}}."
	}
	if utf8.RuneCountInString(r.Notes) > 1000 {
		f["notes"] = "Notiz ist zu lang."
	}

	if len(f) > 0 {
		return &ValidationError{Fields: f}
	}
	return nil
}
