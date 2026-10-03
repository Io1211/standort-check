package backend

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Authentication for the single admin account configured via environment.
//
// Sessions are stateless: the cookie holds "email|expiry" plus an HMAC
// signature made with SESSION_SECRET. This fits serverless (no shared memory,
// no session table). Trade-off: logout only removes the cookie from this
// browser; a copied cookie stays valid until it expires. Rotating
// SESSION_SECRET invalidates all sessions at once.

const (
	sessionCookieName = "sc_session"
	sessionTTL        = 12 * time.Hour
	minSecretLength   = 32
)

type Auth struct {
	email        string // normalized
	passwordHash []byte
	secret       []byte
	now          func() time.Time
}

func NewAuth(cfg Config) *Auth {
	return &Auth{
		email:        NormalizeEmail(cfg.AdminEmail),
		passwordHash: []byte(cfg.AdminPasswordHash),
		secret:       []byte(cfg.SessionSecret),
		now:          time.Now,
	}
}

func (a *Auth) configured() bool {
	return a.email != "" && len(a.passwordHash) > 0 && len(a.secret) >= minSecretLength
}

// checkCredentials always runs bcrypt, so a wrong email takes as long as a
// wrong password and response times don't reveal the admin email.
func (a *Auth) checkCredentials(email, password string) bool {
	emailOK := subtle.ConstantTimeCompare([]byte(NormalizeEmail(email)), []byte(a.email)) == 1
	passwordOK := bcrypt.CompareHashAndPassword(a.passwordHash, []byte(password)) == nil
	return emailOK && passwordOK
}

func (a *Auth) newSession() string {
	expiry := a.now().Add(sessionTTL).Unix()
	payload := base64.RawURLEncoding.EncodeToString([]byte(a.email + "|" + strconv.FormatInt(expiry, 10)))
	return payload + "." + a.sign(payload)
}

// verifySession returns the admin email if the cookie value is authentic,
// not expired and still belongs to the configured admin.
func (a *Auth) verifySession(value string) (string, bool) {
	payload, signature, ok := strings.Cut(value, ".")
	if !ok || !hmac.Equal([]byte(signature), []byte(a.sign(payload))) {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return "", false
	}
	email, expiryStr, ok := strings.Cut(string(raw), "|")
	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if !ok || err != nil || a.now().Unix() > expiry || email != a.email {
		return "", false
	}
	return email, true
}

func (a *Auth) sign(payload string) string {
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *Auth) setCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,                 // not readable from JavaScript
		Secure:   true,                 // session cookies must only travel over HTTPS
		SameSite: http.SameSiteLaxMode, // not sent on cross-site POST/PATCH (CSRF)
	})
}

type ctxKey struct{}

// requireAuth protects admin routes. Admin responses contain personal data,
// so they must never be cached.
func (a *Auth) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		c, err := r.Cookie(sessionCookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Bitte anmelden.")
			return
		}
		email, ok := a.verifySession(c.Value)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Sitzung abgelaufen. Bitte erneut anmelden.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, email)))
	})
}

// requireJSON rejects POST/PUT/PATCH requests that are not JSON. Together
// with SameSite=Lax this blocks cross-site form posts (CSRF): a foreign page
// cannot send application/json to us without a CORS preflight, which we never
// allow. DELETE has no body and needs no check: HTML forms cannot send it,
// and a cross-origin DELETE always requires a preflight.
func requireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasBody := r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch
		if hasBody && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeError(w, http.StatusUnsupportedMediaType, "Content-Type application/json erwartet.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
