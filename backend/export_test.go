package backend

import (
	"bytes"
	"encoding/csv"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestWriteLeadsCSV(t *testing.T) {
	dup := "11111111-1111-1111-1111-111111111111"
	leads := []LeadListItem{{Lead: Lead{
		ID:          "22222222-2222-2222-2222-222222222222",
		FirstName:   "=HYPERLINK(\"http://evil\")",
		LastName:    "Klüpper; Jörg",
		Email:       "joerg@example.com",
		Phone:       "+49 4519988776",
		Street:      "Am Mühlenteich",
		HouseNumber: "7",
		PostalCode:  "01067", // leading zero must survive
		City:        "Groß Grönau",
		ParcelNote:  "Zeile 1\nZeile 2",
		Status:      LeadStatusQualified,
		DuplicateOf: &dup,
		CreatedAt:   time.Date(2026, 10, 3, 7, 30, 0, 0, time.UTC),
	}}}

	var buf bytes.Buffer
	if err := WriteLeadsCSV(&buf, leads); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.HasPrefix(out, "\uFEFF") {
		t.Error("missing UTF-8 BOM")
	}

	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(out, "\uFEFF")))
	r.Comma = ';'
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("CSV does not parse: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want header + 1", len(records))
	}
	row := map[string]string{}
	for i, h := range records[0] {
		row[h] = records[1][i]
	}

	checks := map[string]string{
		"first_name":   `'=HYPERLINK("http://evil")`, // formula neutralized
		"last_name":    "Klüpper; Jörg",              // separator inside value survives
		"phone":        "'+49 4519988776",
		"postal_code":  "01067",
		"city":         "Groß Grönau",
		"parcel_note":  "Zeile 1\nZeile 2", // line break inside a quoted field survives
		"status":       "qualified",
		"duplicate_of": dup,
		"created_at":   "2026-10-03 09:30:00", // German summer time (UTC+2)
	}
	for col, want := range checks {
		if row[col] != want {
			t.Errorf("%s = %q, want %q", col, row[col], want)
		}
	}
}

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"Thomas":   "Thomas",
		"=1+1":     "'=1+1",
		"+49 40":   "'+49 40",
		"-2":       "'-2",
		"@SUM(A1)": "'@SUM(A1)",
		"01067":    "01067",
	}
	for in, want := range cases {
		if got := csvSafe(in); got != want {
			t.Errorf("csvSafe(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseLeadFilter(t *testing.T) {
	f, err := parseLeadFilter(url.Values{})
	if err != nil || f.Sort != "created_at" || !f.Desc {
		t.Errorf("default should be newest first, got %+v, %v", f, err)
	}

	f, err = parseLeadFilter(url.Values{"sort": {"name"}, "order": {"asc"}, "status": {"qualified"}, "hideDuplicates": {"true"}})
	if err != nil || f.Sort != "name" || f.Desc || f.Status != LeadStatusQualified || !f.HideDuplicates {
		t.Errorf("unexpected filter %+v, %v", f, err)
	}

	for _, bad := range []url.Values{
		{"sort": {"email; DROP TABLE leads"}},
		{"status": {"done"}},
	} {
		if _, err := parseLeadFilter(bad); err == nil {
			t.Errorf("expected error for %v", bad)
		}
	}
}

func TestLeadFilterWhereUsesParameters(t *testing.T) {
	f := LeadFilter{Search: "o'brien%", Status: LeadStatusNew, Source: NoValue, Campaign: "dresden"}
	where, args := f.where()
	if strings.Contains(where, "o'brien") || strings.Contains(where, "dresden") {
		t.Errorf("user input must not be part of the SQL: %s", where)
	}
	if !strings.Contains(where, "utm_source IS NULL") {
		t.Errorf("NoValue should filter for missing source: %s", where)
	}
	if args[0] != `%o'brien\%%` {
		t.Errorf("LIKE wildcards in input must be escaped, got %q", args[0])
	}
}
