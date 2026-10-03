package backend

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Normalization is deliberately simple and deterministic: the same input
// always gives the same output, and no guessing (fuzzy matching) is done.

// NormalizeEmail trims and lowercases. We do not strip "+tags" or dots:
// that is provider specific and could merge different people.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(norm.NFC.String(s)))
}

// NormalizeText is used for city names: Unicode NFC (macOS/iOS may send "ü"
// as "u" + combining umlaut), lowercase, collapsed whitespace, ß → ss.
func NormalizeText(s string) string {
	s = norm.NFC.String(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "ß", "ss")
	return strings.Join(strings.Fields(s), " ")
}

// NormalizeStreet is NormalizeText plus one unambiguous German abbreviation:
// a word ending in "str." becomes "...strasse" ("Hauptstr." = "Hauptstraße").
// Other abbreviations are intentionally not expanded.
func NormalizeStreet(s string) string {
	words := strings.Fields(NormalizeText(s))
	for i, w := range words {
		if strings.HasSuffix(w, "str.") {
			words[i] = strings.TrimSuffix(w, ".") + "asse"
		}
	}
	return strings.Join(words, " ")
}

// NormalizeHouseNumber lowercases and removes all whitespace: "14 A" = "14a".
func NormalizeHouseNumber(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), ""))
}

// NormalizePostalCode only trims. Postal codes stay strings ("01067").
func NormalizePostalCode(s string) string {
	return strings.TrimSpace(s)
}

// ComposePhone joins the dropdown country code and the national number typed
// in the form. A leading trunk "0" is dropped: "+49" + "0170 1234567" →
// raw "+49 170 1234567", normalized "+491701234567". Expects validated input.
func ComposePhone(countryCode, national string) (raw, normalized string) {
	digits := strings.TrimLeft(strings.Join(strings.Fields(national), ""), "0")
	return countryCode + " " + digits, countryCode + digits
}

// NormalizePhone converts German phone numbers to an E.164-like form so that
// "+49 40 / 123 456", "004940123456" and "040 123456" all become
// "+4940123456". Numbers without any prefix are returned as plain digits.
func NormalizePhone(s string) string {
	// "+49 (0)40 ..." – the (0) is a common, technically wrong trunk prefix.
	s = strings.ReplaceAll(strings.TrimSpace(s), "(0)", "")

	var b strings.Builder
	for i, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == '+' && i == 0 {
			b.WriteRune(r)
		}
	}
	p := b.String()

	switch {
	case strings.HasPrefix(p, "00"):
		p = "+" + p[2:]
	case strings.HasPrefix(p, "0"):
		p = "+49" + p[1:]
	}
	// "+49 0170 …": country code plus the national trunk "0" – drop the 0.
	if strings.HasPrefix(p, "+490") {
		p = "+49" + p[4:]
	}
	return p
}
