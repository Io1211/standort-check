package backend

import (
	"encoding/csv"
	"io"
	"strings"
	"time"
	_ "time/tzdata" // embed tz database: serverless images may not ship one
)

var berlin = mustLoadLocation("Europe/Berlin")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

var csvHeader = []string{
	"id", "created_at", "status", "duplicate_of",
	"first_name", "last_name", "email", "phone",
	"street", "house_number", "postal_code", "city", "parcel_note",
	"utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term",
	"gclid", "fbclid", "referrer",
}

// WriteLeadsCSV writes leads in a format that opens correctly in German
// Excel: UTF-8 with BOM (otherwise umlauts break), ";" as separator (Excel
// uses the locale's list separator) and times in German local time.
// encoding/csv takes care of quoting values with ";", quotes or line breaks.
func WriteLeadsCSV(w io.Writer, leads []LeadListItem) error {
	if _, err := io.WriteString(w, "\uFEFF"); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	cw.UseCRLF = true

	if err := cw.Write(csvHeader); err != nil {
		return err
	}
	for _, l := range leads {
		dup := ""
		if l.DuplicateOf != nil {
			dup = *l.DuplicateOf
		}
		record := []string{
			l.ID, l.CreatedAt.In(berlin).Format("2006-01-02 15:04:05"), string(l.Status), dup,
			l.FirstName, l.LastName, l.Email, l.Phone,
			l.Street, l.HouseNumber, l.PostalCode, l.City, l.ParcelNote,
			l.UTMSource, l.UTMMedium, l.UTMCampaign, l.UTMContent, l.UTMTerm,
			l.GCLID, l.FBCLID, l.Referrer,
		}
		for i := range record {
			record[i] = csvSafe(record[i])
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// csvSafe prevents CSV/formula injection: Excel executes cells starting with
// = + - @ (a lead could enter "=HYPERLINK(...)" as their name). Such values
// get a leading apostrophe. This also keeps "+49 170…" from being turned
// into a number or formula. Plain negative numbers never occur in our data.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	if strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func exportFilename(now time.Time) string {
	return "standort-check-leads-" + now.In(berlin).Format("2006-01-02") + ".csv"
}
