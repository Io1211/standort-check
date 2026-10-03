package backend

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const testPassword = "correct horse battery"

func testAuth(t *testing.T) *Auth {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return NewAuth(Config{
		AdminEmail:        "Sales@Example.com",
		AdminPasswordHash: string(hash),
		SessionSecret:     strings.Repeat("s", 32),
	})
}

func TestCheckCredentials(t *testing.T) {
	a := testAuth(t)
	cases := []struct {
		email, password string
		want            bool
	}{
		{"sales@example.com", testPassword, true},
		{" SALES@example.com ", testPassword, true}, // email is normalized
		{"sales@example.com", "wrong", false},
		{"other@example.com", testPassword, false},
		{"", "", false},
	}
	for _, tc := range cases {
		if got := a.checkCredentials(tc.email, tc.password); got != tc.want {
			t.Errorf("checkCredentials(%q, %q) = %v, want %v", tc.email, tc.password, got, tc.want)
		}
	}
}

func TestSession(t *testing.T) {
	a := testAuth(t)
	now := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return now }

	valid := a.newSession()
	if email, ok := a.verifySession(valid); !ok || email != "sales@example.com" {
		t.Fatalf("fresh session rejected")
	}

	payload, sig, _ := strings.Cut(valid, ".")
	tampered := payload + "x." + sig
	if _, ok := a.verifySession(tampered); ok {
		t.Error("tampered payload accepted")
	}
	if _, ok := a.verifySession(payload + "." + strings.Repeat("A", len(sig))); ok {
		t.Error("forged signature accepted")
	}
	if _, ok := a.verifySession("garbage"); ok {
		t.Error("garbage accepted")
	}

	// Same cookie signed with another secret (e.g. after rotation).
	other := testAuth(t)
	other.secret = []byte(strings.Repeat("t", 32))
	if _, ok := other.verifySession(valid); ok {
		t.Error("session from another secret accepted")
	}

	a.now = func() time.Time { return now.Add(sessionTTL + time.Second) }
	if _, ok := a.verifySession(valid); ok {
		t.Error("expired session accepted")
	}
}

func TestNotConfiguredWithShortSecret(t *testing.T) {
	a := NewAuth(Config{AdminEmail: "a@b.de", AdminPasswordHash: "x", SessionSecret: "short"})
	if a.configured() {
		t.Error("a short SESSION_SECRET must not count as configured")
	}
}

// --- HTTP level ---

func adminServer(t *testing.T) http.Handler {
	t.Helper()
	return (&Server{auth: testAuth(t)}).routes()
}

func do(h http.Handler, method, path, body, contentType string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestLoginFlow(t *testing.T) {
	h := adminServer(t)

	if rec := do(h, "GET", "/api/auth/me", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("me without cookie: %d, want 401", rec.Code)
	}

	rec := do(h, "POST", "/api/auth/login", `{"email":"sales@example.com","password":"wrong"}`, "application/json")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d, want 401", rec.Code)
	}

	rec = do(h, "POST", "/api/auth/login", `{"email":"sales@example.com","password":"`+testPassword+`"}`, "application/json")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d, want 200", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session cookie: %+v", cookies)
	}

	if rec := do(h, "GET", "/api/auth/me", "", "", cookies[0]); rec.Code != http.StatusOK {
		t.Fatalf("me with cookie: %d, want 200", rec.Code)
	}
}

func TestSessionCookiesAreSecure(t *testing.T) {
	a := NewAuth(Config{})
	for _, maxAge := range []int{int(sessionTTL.Seconds()), -1} {
		rec := httptest.NewRecorder()
		a.setCookie(rec, "", maxAge)
		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].Secure {
			t.Fatalf("maxAge=%d: expected Secure session cookie, got %+v", maxAge, cookies)
		}
	}
}

func TestLoginRequiresJSON(t *testing.T) {
	// A cross-site HTML form can only send form-encoded bodies.
	rec := do(adminServer(t), "POST", "/api/auth/login", "email=a&password=b", "application/x-www-form-urlencoded")
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("form post: %d, want 415", rec.Code)
	}
}

func TestAdminRoutesRequireAuth(t *testing.T) {
	h := adminServer(t)
	routes := []struct{ method, path string }{
		{"GET", "/api/leads"},
		{"GET", "/api/leads/export"},
		{"GET", "/api/leads/stats"},
		{"GET", "/api/leads/00000000-0000-0000-0000-000000000000"},
		{"PATCH", "/api/leads/00000000-0000-0000-0000-000000000000/status"},
		{"DELETE", "/api/leads/00000000-0000-0000-0000-000000000000"},
		{"GET", "/api/leads/timeseries"},
		{"GET", "/api/leads/filter-options"},
	}
	for _, rt := range routes {
		if rec := do(h, rt.method, rt.path, `{"status":"new"}`, "application/json"); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without session: %d, want 401", rt.method, rt.path, rec.Code)
		}
	}
}

func TestInvalidLeadIDIs404(t *testing.T) {
	a := testAuth(t)
	h := (&Server{auth: a}).routes()
	cookie := &http.Cookie{Name: sessionCookieName, Value: a.newSession()}

	// Returns before the database is touched (no repo in this test server).
	if rec := do(h, "GET", "/api/leads/not-a-uuid", "", "", cookie); rec.Code != http.StatusNotFound {
		t.Errorf("GET invalid id: %d, want 404", rec.Code)
	}
	if rec := do(h, "PATCH", "/api/leads/not-a-uuid/status", `{"status":"new"}`, "application/json", cookie); rec.Code != http.StatusNotFound {
		t.Errorf("PATCH invalid id: %d, want 404", rec.Code)
	}
	if rec := do(h, "PATCH", "/api/leads/00000000-0000-0000-0000-000000000000/status", `{"status":"done"}`, "application/json", cookie); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("PATCH invalid status: %d, want 422", rec.Code)
	}
	// DELETE carries no body, so it must not be rejected for a missing Content-Type.
	if rec := do(h, "DELETE", "/api/leads/not-a-uuid", "", "", cookie); rec.Code != http.StatusNotFound {
		t.Errorf("DELETE without Content-Type: %d, want 404", rec.Code)
	}
}
