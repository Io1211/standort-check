package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These tests cover the paths that return before the database is touched,
// so no repository is needed.
func newTestServer() http.Handler {
	s := &Server{leads: NewLeadService(nil)}
	return s.routes()
}

func post(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/leads", strings.NewReader(body))
	rec := httptest.NewRecorder()
	newTestServer().ServeHTTP(rec, req)
	return rec
}

func TestCreateLead_InvalidJSON(t *testing.T) {
	if rec := post(t, `{not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateLead_ValidationErrorReturnsFields(t *testing.T) {
	rec := post(t, `{"firstName":"Thomas","email":"kaputt"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	var body struct {
		Fields map[string]string `json:"fields"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"lastName", "email", "phone", "street", "postalCode", "city", "consent"} {
		if _, ok := body.Fields[f]; !ok {
			t.Errorf("expected field error for %q, got %v", f, body.Fields)
		}
	}
}

func TestCreateLead_HoneypotLooksLikeSuccess(t *testing.T) {
	// A bot fills every field, including the hidden "website" field. If this
	// reached the repository the test would panic on the nil repo.
	rec := post(t, `{"firstName":"Bot","lastName":"Bot","email":"bot@example.com","phone":"040123456",
		"street":"x","postalCode":"12345","city":"x","consent":true,"website":"http://spam.example"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
}

func TestCreateLead_BodyTooLarge(t *testing.T) {
	big := `{"firstName":"` + strings.Repeat("a", 64<<10) + `"}`
	if rec := post(t, big); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
