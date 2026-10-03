package backend

// Duplicate rule (business decision):
//
//	A new lead is a possible duplicate of an existing lead if
//	  the normalized plot address is the same
//	  AND (the normalized email is the same OR the normalized phone is the same).
//
// Address alone is not enough (neighbours or co-owners may be different
// customers). Email alone is not enough (one customer may ask about several
// plots). Phone counts as contact because the same person often uses
// different email addresses but rarely different phone numbers – see case
// examples #1 and #4 ("+49 40 / 123 456" vs "004940123456").
//
// Names are ignored: spelling varies and they are not identifying.
//
// The database only pre-selects leads with the same normalized address (fast,
// indexed). The decision itself is made here, so the rule exists exactly once
// and is unit-testable without a database.

func sameAddress(a, b Lead) bool {
	return a.PostalCode == b.PostalCode &&
		a.StreetNormalized == b.StreetNormalized &&
		a.HouseNumberNormalized == b.HouseNumberNormalized &&
		a.CityNormalized == b.CityNormalized
}

func sameContact(a, b Lead) bool {
	sameEmail := a.EmailNormalized != "" && a.EmailNormalized == b.EmailNormalized
	samePhone := a.PhoneNormalized != "" && a.PhoneNormalized == b.PhoneNormalized
	return sameEmail || samePhone
}

// IsDuplicate expects both leads to be normalized.
func IsDuplicate(incoming, existing Lead) bool {
	return sameAddress(incoming, existing) && sameContact(incoming, existing)
}

// pickDuplicate returns the first match in candidates. The repository returns
// candidates oldest first, so duplicate_of always points at the oldest match.
func pickDuplicate(incoming Lead, candidates []Lead) *Lead {
	for i := range candidates {
		if IsDuplicate(incoming, candidates[i]) {
			return &candidates[i]
		}
	}
	return nil
}
