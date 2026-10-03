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

var postalCodeRe = regexp.MustCompile(`^[0-9]{5}$`)

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
		digits := strings.TrimPrefix(NormalizePhone(req.Phone), "+")
		if len(digits) < 7 || len(digits) > 15 {
			f["phone"] = "Bitte eine gültige Telefonnummer angeben."
		}
	}

	if required("postalCode", req.PostalCode, "Bitte Postleitzahl angeben.") &&
		!postalCodeRe.MatchString(NormalizePostalCode(req.PostalCode)) {
		f["postalCode"] = "Die Postleitzahl muss aus 5 Ziffern bestehen."
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
