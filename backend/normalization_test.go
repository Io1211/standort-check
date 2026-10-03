package backend

import "testing"

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		" TEST@Example.COM ": "test@example.com",
		"a.b+tag@web.de":     "a.b+tag@web.de", // tags/dots are kept on purpose
	}
	for in, want := range cases {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeStreet(t *testing.T) {
	cases := map[string]string{
		"  Hauptstraße   ":   "hauptstrasse",
		"Hauptstrasse":       "hauptstrasse",
		"Hauptstr.":          "hauptstrasse",
		"HAUPTSTRASSE":       "hauptstrasse",
		"Am   Mühlenteich":   "am mühlenteich",
		"Karl-Marx-Str.":     "karl-marx-strasse",
		"Lange Str.":         "lange strasse",
		"Hauptstr":           "hauptstr",     // no dot → not treated as abbreviation
		"Mühlenweg":         "mühlenweg",    // decomposed ü (macOS) → NFC
		"Am Markt Pl.":       "am markt pl.", // other abbreviations stay as typed
		"\tOsterstraße\n 88": "osterstrasse 88",
	}
	for in, want := range cases {
		if got := NormalizeStreet(in); got != want {
			t.Errorf("NormalizeStreet(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeCity(t *testing.T) {
	cases := map[string]string{
		"  Bad   Neustadt ": "bad neustadt",
		"Groß Grönau":       "gross grönau",
		"DRESDEN":           "dresden",
	}
	for in, want := range cases {
		if got := NormalizeText(in); got != want {
			t.Errorf("NormalizeText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeHouseNumber(t *testing.T) {
	cases := map[string]string{
		" 14A ": "14a",
		"14 a":  "14a",
		"3-5":   "3-5",
		"":      "",
	}
	for in, want := range cases {
		if got := NormalizeHouseNumber(in); got != want {
			t.Errorf("NormalizeHouseNumber(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizePostalCode(t *testing.T) {
	if got := NormalizePostalCode(" 01067 "); got != "01067" {
		t.Errorf("leading zero must survive, got %q", got)
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"+49 40 / 123 456":  "+4940123456", // case example #1
		"004940123456":      "+4940123456", // case example #4
		"040 123456":        "+4940123456",
		"+49 (0)40 123456":  "+4940123456",
		"0170 5551234":      "+491705551234",
		"040 55512345":      "+494055512345",
		"+43 1 234567":      "+431234567", // foreign numbers keep their country code
		"0041-44-1234567":   "+41441234567",
		"1234567":           "1234567", // no prefix → digits only, no guessing
		" +49-40-123-456 ":  "+4940123456",
		"tel: 040 / 123456": "+4940123456",
	}
	for in, want := range cases {
		if got := NormalizePhone(in); got != want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}
