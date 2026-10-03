package backend

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError maps JSON field names to user-facing (German) messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validation failed" }

// German postal codes only (the service covers plots in Germany): five
// digits, and no German postal code starts with "00".
var postalCodeRe = regexp.MustCompile(`^(0[1-9]|[1-9][0-9])[0-9]{3}$`)

// Validate checks a request. The frontend validates too, but only for
// usability – this is the check that protects data integrity.
func Validate(req CreateLeadRequest) *ValidationError {
	f := map[string]string{}

	required := func(field, value, msg string) bool {
		if strings.TrimSpace(value) == "" {
			f[field] = msg
			return false
		}
		return true
	}
	maxLen := func(field, value string, n int) {
		if _, set := f[field]; !set && utf8.RuneCountInString(strings.TrimSpace(value)) > n {
			f[field] = "Eingabe ist zu lang."
		}
	}

	required("firstName", req.FirstName, "Bitte Vornamen angeben.")
	required("lastName", req.LastName, "Bitte Nachnamen angeben.")
	required("street", req.Street, "Bitte Straße angeben.")
	required("city", req.City, "Bitte Ort angeben.")

	if required("email", req.Email, "Bitte E-Mail-Adresse angeben.") && !validEmail(req.Email) {
		f["email"] = "Bitte eine gültige E-Mail-Adresse angeben."
	}

	if required("phone", req.Phone, "Bitte Telefonnummer angeben.") {
		if msg := validatePhone(req.PhoneCountryCode, req.Phone); msg != "" {
			f["phone"] = msg
		}
	}

	if required("postalCode", req.PostalCode, "Bitte Postleitzahl angeben.") &&
		!postalCodeRe.MatchString(NormalizePostalCode(req.PostalCode)) {
		f["postalCode"] = "Bitte eine gültige deutsche Postleitzahl angeben (5 Ziffern)."
	}

	if !req.Consent {
		f["consent"] = "Bitte stimmen Sie der Datenverarbeitung zu."
	}

	maxLen("firstName", req.FirstName, 100)
	maxLen("lastName", req.LastName, 100)
	maxLen("email", req.Email, 254)
	maxLen("phone", req.Phone, 40)
	maxLen("street", req.Street, 150)
	maxLen("houseNumber", req.HouseNumber, 20)
	maxLen("city", req.City, 100)
	maxLen("parcelNote", req.ParcelNote, 500)

	if len(f) > 0 {
		return &ValidationError{Fields: f}
	}
	return nil
}

// Minimum/maximum length of the national number without leading zeros.
// German numbers have at least 7 digits (area code + subscriber number);
// E.164 allows at most 15 digits including the country code.
const (
	minPhoneDigits = 7
	maxPhoneDigits = 15
)

var countryCodeRe = regexp.MustCompile(`^\+[1-9][0-9]{0,3}$`)

// validatePhone returns a user-facing error message or "".
//
// With a country code (the form's dropdown) the number must consist of
// digits only. Without one (API clients) a full number in any common format
// is accepted and normalized by NormalizePhone.
func validatePhone(countryCode, phone string) string {
	const invalid = "Bitte eine gültige Telefonnummer angeben (mind. 7 Ziffern)."

	if countryCode == "" {
		digits := strings.TrimPrefix(NormalizePhone(phone), "+")
		if len(digits) < minPhoneDigits || len(digits) > maxPhoneDigits {
			return invalid
		}
		return ""
	}

	if !countryCodeRe.MatchString(countryCode) {
		return "Bitte eine gültige Ländervorwahl wählen."
	}
	national := strings.Join(strings.Fields(phone), "")
	for _, r := range national {
		if r < '0' || r > '9' {
			return "Die Telefonnummer darf nur Ziffern enthalten."
		}
	}
	nsn := strings.TrimLeft(national, "0")
	total := len(countryCode) - 1 + len(nsn)
	if len(nsn) < minPhoneDigits || total > maxPhoneDigits {
		return invalid
	}
	return ""
}

// validEmail accepts plain addresses like "a@b.de" and rejects display-name
// forms ("Max <a@b.de>") and domains without a dot.
func validEmail(s string) bool {
	s = strings.TrimSpace(s)
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}
	at := strings.LastIndex(s, "@")
	return at > 0 && strings.Contains(s[at+1:], ".")
}
