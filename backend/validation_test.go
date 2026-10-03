package backend

import "testing"

func validRequest() CreateLeadRequest {
	return CreateLeadRequest{
		FirstName:  "Thomas",
		LastName:   "Ahrens",
		Email:      "thomas@example.com",
		Phone:      "+49 40 123456",
		Street:     "Hauptstraße",
		PostalCode: "01067",
		City:       "Dresden",
		Consent:    true,
	}
}

func TestValidate_ValidRequest(t *testing.T) {
	if err := Validate(validRequest()); err != nil {
		t.Fatalf("expected no error, got %v", err.Fields)
	}
}

func TestValidate_HouseNumberIsOptional(t *testing.T) {
	req := validRequest()
	req.HouseNumber = ""
	req.ParcelNote = "Flurstück 123/4, Gemarkung Klotzsche"
	if err := Validate(req); err != nil {
		t.Fatalf("plot without house number must be valid, got %v", err.Fields)
	}
}

func TestValidate_Errors(t *testing.T) {
	cases := []struct {
		name   string
		modify func(*CreateLeadRequest)
		field  string
	}{
		{"missing first name", func(r *CreateLeadRequest) { r.FirstName = "  " }, "firstName"},
		{"missing last name", func(r *CreateLeadRequest) { r.LastName = "" }, "lastName"},
		{"missing email", func(r *CreateLeadRequest) { r.Email = "" }, "email"},
		{"invalid email", func(r *CreateLeadRequest) { r.Email = "thomas@" }, "email"},
		{"email without dot in domain", func(r *CreateLeadRequest) { r.Email = "thomas@localhost" }, "email"},
		{"email with display name", func(r *CreateLeadRequest) { r.Email = "Thomas <t@example.com>" }, "email"},
		{"missing phone", func(r *CreateLeadRequest) { r.Phone = "" }, "phone"},
		{"phone too short", func(r *CreateLeadRequest) { r.Phone = "123" }, "phone"},
		{"phone without digits", func(r *CreateLeadRequest) { r.Phone = "keine" }, "phone"},
		{"missing street", func(r *CreateLeadRequest) { r.Street = "" }, "street"},
		{"missing city", func(r *CreateLeadRequest) { r.City = "" }, "city"},
		{"missing postal code", func(r *CreateLeadRequest) { r.PostalCode = "" }, "postalCode"},
		{"postal code with 4 digits", func(r *CreateLeadRequest) { r.PostalCode = "1067" }, "postalCode"},
		{"postal code with letters", func(r *CreateLeadRequest) { r.PostalCode = "0106A" }, "postalCode"},
		{"postal code starting with 00", func(r *CreateLeadRequest) { r.PostalCode = "00123" }, "postalCode"},
		{"Austrian postal code", func(r *CreateLeadRequest) { r.PostalCode = "6020" }, "postalCode"},
		{"no consent", func(r *CreateLeadRequest) { r.Consent = false }, "consent"},
		{"house number too long", func(r *CreateLeadRequest) { r.HouseNumber = "123456789012345678901" }, "houseNumber"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validRequest()
			tc.modify(&req)
			err := Validate(req)
			if err == nil {
				t.Fatalf("expected validation error for %s", tc.field)
			}
			if _, ok := err.Fields[tc.field]; !ok {
				t.Fatalf("expected error on %q, got %v", tc.field, err.Fields)
			}
		})
	}
}

func TestValidate_PostalCodeIsTrimmed(t *testing.T) {
	req := validRequest()
	req.PostalCode = " 01067 "
	if err := Validate(req); err != nil {
		t.Fatalf("expected trimmed postal code to be valid, got %v", err.Fields)
	}
}

func TestValidate_PhoneWithCountryCode(t *testing.T) {
	cases := []struct {
		code, phone string
		wantError   bool
	}{
		{"+49", "1701234567", false},
		{"+49", "0170 1234567", false}, // trunk 0 is allowed and dropped
		{"+43", "69912112010", false},
		{"+49", "30123456", false}, // Berlin landline, 8 digits
		{"+49", "123456", true},    // too short
		{"+49", "0123456", true},   // 6 digits after dropping the 0
		{"+49", "0170-1234567", true},
		{"+49", "+49 170 1234567", true}, // full number in national field
		{"+49", "abc1234567", true},
		{"49", "1701234567", true},        // country code without +
		{"+0", "1701234567", true},        // invalid country code
		{"+49", "1234567890123456", true}, // longer than E.164 allows
	}
	for _, tc := range cases {
		req := validRequest()
		req.PhoneCountryCode, req.Phone = tc.code, tc.phone
		err := Validate(req)
		gotError := false
		if err != nil {
			_, gotError = err.Fields["phone"]
		}
		if gotError != tc.wantError {
			t.Errorf("code=%q phone=%q: phone error = %v, want %v", tc.code, tc.phone, gotError, tc.wantError)
		}
	}
}
