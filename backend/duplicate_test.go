package backend

import (
	"fmt"
	"testing"
	"time"
)

// lead builds a normalized Lead from raw input, exactly like production does.
func lead(email, phone, street, houseNumber, postalCode, city string) Lead {
	return buildLead(CreateLeadRequest{
		FirstName: "Test", LastName: "Test",
		Email: email, Phone: phone,
		Street: street, HouseNumber: houseNumber, PostalCode: postalCode, City: city,
		Consent: true,
	}, time.Time{})
}

func TestIsDuplicate(t *testing.T) {
	existing := lead("test@example.com", "+49 40 123456", "Hauptstraße", "14", "01067", "Dresden")

	cases := []struct {
		name     string
		incoming Lead
		want     bool
	}{
		{"same email and address despite case/whitespace",
			lead(" TEST@example.com", "0170 999999", " Hauptstraße ", "14", "01067", " Dresden"), true},
		{"same email, different house number",
			lead("test@example.com", "0170 999999", "Hauptstraße", "16", "01067", "Dresden"), false},
		{"same address, different email and different phone",
			lead("other@example.com", "0170 999999", "Hauptstraße", "14", "01067", "Dresden"), false},
		{"same address, different email, same phone in other format",
			lead("other@example.com", "004940123456", "Hauptstraße", "14", "01067", "Dresden"), true},
		{"same email, phone differs → still duplicate",
			lead("test@example.com", "040 999999", "Hauptstraße", "14", "01067", "Dresden"), true},
		{"street abbreviation and ß/ss are equal",
			lead("test@example.com", "", "HAUPTSTR.", "14", "01067", "Dresden"), true},
		{"house number spacing/case is equal",
			lead("test@example.com", "", "Hauptstraße", " 14 ", "01067", "dresden"), true},
		{"same street in a different town (other postal code)",
			lead("test@example.com", "+49 40 123456", "Hauptstraße", "14", "01069", "Dresden"), false},
		{"house number missing vs. given",
			lead("test@example.com", "+49 40 123456", "Hauptstraße", "", "01067", "Dresden"), false},
		{"different suffix 14 vs 14a",
			lead("test@example.com", "+49 40 123456", "Hauptstraße", "14a", "01067", "Dresden"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsDuplicate(tc.incoming, existing); got != tc.want {
				t.Errorf("IsDuplicate = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsDuplicate_EmptyContactNeverMatches(t *testing.T) {
	a := lead("", "", "Hauptstraße", "14", "01067", "Dresden")
	b := lead("", "", "Hauptstraße", "14", "01067", "Dresden")
	if IsDuplicate(a, b) {
		t.Error("empty email/phone must not count as 'same contact'")
	}
}

func TestPickDuplicate_PointsToOldestMatch(t *testing.T) {
	first := lead("a@example.com", "040 111111", "Lindenweg", "3", "12345", "Neustadt")
	first.ID = "first"
	second := lead("a@example.com", "040 222222", "Lindenweg", "3", "12345", "Neustadt")
	second.ID = "second"

	incoming := lead("a@example.com", "040 333333", "Lindenweg", "3", "12345", "Neustadt")
	got := pickDuplicate(incoming, []Lead{first, second})
	if got == nil || got.ID != "first" {
		t.Fatalf("expected oldest match 'first', got %+v", got)
	}
}

// The five example requests from the case description. Emails are not given
// in the case, so #1 and #4 deliberately use different addresses: they must
// still be recognized as the same person via the phone number.
func TestCaseExamples(t *testing.T) {
	examples := []CreateLeadRequest{
		{FirstName: "Thomas", LastName: "Ahrens", Email: "thomas.ahrens@example.com", Phone: "+49 40 / 123 456",
			Street: "Hauptstraße", HouseNumber: "14", PostalCode: "01067", City: "Dresden"},
		{FirstName: "Marion", LastName: "Beckmann", Email: "marion@example.com", Phone: "0170 5551234",
			Street: "Lindenweg", HouseNumber: "3", PostalCode: "", City: "Neustadt"},
		{FirstName: "Kai", LastName: "Ruthenberg", Email: "kai@example.com", Phone: "040 55512345",
			Street: "Osterstraße", HouseNumber: "88", PostalCode: "22765", City: "Hamburg"},
		{FirstName: "Thomas", LastName: "Ahrens", Email: "t.ahrens@web.de", Phone: "004940123456",
			Street: "Hauptstraße", HouseNumber: "14", PostalCode: "01067", City: "Dresden"},
		{FirstName: "Jörg", LastName: "Klüpper", Email: "joerg@example.com", Phone: "0451 9988776",
			Street: "Am Mühlenteich", HouseNumber: "7", PostalCode: "23627", City: "Groß Grönau"},
	}

	var stored []Lead
	duplicateOf := map[int]string{}

	for i, req := range examples {
		req.Consent = true
		if verr := Validate(req); verr != nil {
			// #2 "Lindenweg 3, Neustadt" has no postal code – there are many
			// Neustadts in Germany, so it is rejected and the user is asked.
			if i != 1 {
				t.Fatalf("example #%d unexpectedly invalid: %v", i+1, verr.Fields)
			}
			if _, ok := verr.Fields["postalCode"]; !ok {
				t.Fatalf("example #2 should fail on postalCode, got %v", verr.Fields)
			}
			continue
		}
		l := buildLead(req, time.Time{})
		l.ID = fmt.Sprintf("#%d", i+1)
		if dup := pickDuplicate(l, stored); dup != nil {
			duplicateOf[i+1] = dup.ID
		}
		stored = append(stored, l)
	}

	want := map[int]string{4: "#1"}
	if fmt.Sprint(duplicateOf) != fmt.Sprint(want) {
		t.Fatalf("duplicates = %v, want %v", duplicateOf, want)
	}
}
