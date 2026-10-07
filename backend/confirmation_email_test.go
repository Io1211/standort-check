package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type emailRoundTrip func(*http.Request) (*http.Response, error)

func (f emailRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// sendersJSON mimics Brevo's GET /v3/senders answer.
func sendersJSON(email string, active bool) *http.Response {
	body := `{"senders":[{"id":1,"name":"Team","email":"` + email + `","active":` + map[bool]string{true: "true", false: "false"}[active] + `,"ips":[]}]}`
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}

func TestBrevoConfirmationPayload(t *testing.T) {
	m := newConfirmationSender(Config{BrevoAPIKey: "test-key", EmailFrom: "sender@example.com"}).(*brevoMailer)
	var calls []string
	m.client = &http.Client{Transport: emailRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Header.Get("api-key") != "test-key" {
			t.Fatal("missing authentication header")
		}
		if r.Method == http.MethodGet && r.URL.String() == "https://api.brevo.com/v3/senders" {
			// Brevo stores the address as typed in the account; matching is case-insensitive.
			return sendersJSON("Sender@Example.com", true), nil
		}
		if r.Method != http.MethodPost || r.URL.String() != "https://api.brevo.com/v3/smtp/email" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatal("missing JSON header")
		}
		var body struct {
			Sender  emailContact   `json:"sender"`
			To      []emailContact `json:"to"`
			ReplyTo emailContact   `json:"replyTo"`
			Subject string         `json:"subject"`
			Text    string         `json:"textContent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Sender.Email != "sender@example.com" || body.ReplyTo.Email != "sender@example.com" || len(body.To) != 1 || body.To[0].Email != "thomas@example.com" {
			t.Fatalf("unexpected sender/recipient: %+v", body)
		}
		if !strings.Contains(body.Subject, "Bestätigung") || !strings.Contains(body.Text, "Thomas Ahrens") || !strings.Contains(body.Text, "Standort") {
			t.Fatalf("missing German confirmation content: %+v", body)
		}
		return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"messageId":"test"}`))}, nil
	})}
	if err := m.SendConfirmation(context.Background(), buildLead(validRequest())); err != nil {
		t.Fatal(err)
	}
	// The second send within the TTL must reuse the verified sender.
	if err := m.SendConfirmation(context.Background(), buildLead(validRequest())); err != nil {
		t.Fatal(err)
	}
	want := []string{"GET /v3/senders", "POST /v3/smtp/email", "POST /v3/smtp/email"}
	if strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

// Brevo answers HTTP 201 for an unverified sender and only rejects the mail
// afterwards. The app must therefore never report such a mail as sent.
func TestBrevoUnverifiedSenderIsNotSent(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response func() *http.Response
	}{
		{"sender missing", func() *http.Response { return sendersJSON("other@example.com", true) }},
		{"sender not yet verified", func() *http.Response { return sendersJSON("sender@example.com", false) }},
		{"sender list unavailable", func() *http.Response {
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("private recipient details"))}
		}},
		{"sender list malformed", func() *http.Response {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("<html>"))}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newConfirmationSender(Config{BrevoAPIKey: "test-key", EmailFrom: "sender@example.com"}).(*brevoMailer)
			m.client = &http.Client{Transport: emailRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodPost {
					t.Fatal("mail must not be sent through an unverified sender")
				}
				return tc.response(), nil
			})}
			err := m.SendConfirmation(context.Background(), buildLead(validRequest()))
			if err == nil || strings.Contains(err.Error(), "private") {
				t.Fatalf("unsafe or missing error: %v", err)
			}
			if !m.verifiedAt.IsZero() {
				t.Fatal("a failed check must not be cached as verified")
			}
		})
	}
}

func TestBrevoSenderCheckExpires(t *testing.T) {
	m := newConfirmationSender(Config{BrevoAPIKey: "test-key", EmailFrom: "sender@example.com"}).(*brevoMailer)
	checks := 0
	m.client = &http.Client{Transport: emailRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodGet {
			checks++
			return sendersJSON("sender@example.com", true), nil
		}
		return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}
	m.verifiedAt = time.Now().Add(-senderCheckTTL - time.Minute) // stale positive result
	if err := m.SendConfirmation(context.Background(), buildLead(validRequest())); err != nil {
		t.Fatal(err)
	}
	if checks != 1 {
		t.Fatalf("stale sender check must be repeated, got %d checks", checks)
	}
}

func TestBrevoConfirmationFailures(t *testing.T) {
	for _, status := range []int{400, 401, 429, 500} {
		m := newConfirmationSender(Config{BrevoAPIKey: "test-key", EmailFrom: "sender@example.com"}).(*brevoMailer)
		m.client = &http.Client{Transport: emailRoundTrip(func(r *http.Request) (*http.Response, error) {
			if r.Method == http.MethodGet {
				return sendersJSON("sender@example.com", true), nil
			}
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("private recipient details"))}, nil
		})}
		err := m.SendConfirmation(context.Background(), buildLead(validRequest()))
		if err == nil || strings.Contains(err.Error(), "private") {
			t.Fatalf("status=%d: unsafe or missing error: %v", status, err)
		}
	}
	for _, cfg := range []Config{{}, {BrevoAPIKey: "test-key"}, {EmailFrom: "sender@example.com"}} {
		if newConfirmationSender(cfg) != nil {
			t.Fatal("incomplete configuration must disable mail")
		}
	}
}

type confirmationTestStore struct {
	saved bool
	err   error
}

func (s *confirmationTestStore) CreateLead(_ context.Context, lead Lead) (Lead, error) {
	if s.err != nil {
		return Lead{}, s.err
	}
	s.saved = true
	lead.ID = "saved-lead"
	return lead, nil
}

type confirmationTestSender struct {
	t     *testing.T
	store *confirmationTestStore
	calls int
	err   error
}

func (m *confirmationTestSender) SendConfirmation(ctx context.Context, lead Lead) error {
	m.calls++
	if !m.store.saved || lead.ID != "saved-lead" {
		m.t.Fatal("mail must follow a successful save")
	}
	if lead.Email != "thomas@example.com" {
		m.t.Fatalf("unexpected recipient: %s", lead.Email)
	}
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > confirmationTimeout || ctx.Err() != nil {
		m.t.Fatal("mail must have its own bounded context")
	}
	return m.err
}

func TestCreateLeadConfirmationFlow(t *testing.T) {
	for _, tc := range []struct {
		name              string
		modify            func(*CreateLeadRequest)
		storeErr, mailErr error
		disabled          bool
		status, calls     int
		sent              bool
	}{
		{name: "sent after save", status: 201, calls: 1, sent: true},
		{name: "mail failure preserves submission", mailErr: errors.New("provider unavailable"), status: 201, calls: 1},
		{name: "not configured", disabled: true, status: 201},
		{name: "save failed", storeErr: errors.New("database unavailable"), status: 500},
		{name: "invalid", modify: func(r *CreateLeadRequest) { r.Email = "invalid" }, status: 422},
		{name: "honeypot", modify: func(r *CreateLeadRequest) { r.Website = "spam" }, status: 201},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &confirmationTestStore{err: tc.storeErr}
			mailer := &confirmationTestSender{t: t, store: store, err: tc.mailErr}
			server := &Server{leads: NewLeadService(store), mailer: mailer}
			if tc.disabled {
				server.mailer = nil
			}
			input := validRequest()
			if tc.modify != nil {
				tc.modify(&input)
			}
			body, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewReader(body))
			// A disconnected browser must not suppress mail for a saved lead.
			ctx, cancel := context.WithCancel(r.Context())
			cancel()
			rec := httptest.NewRecorder()
			server.handleCreateLead(rec, r.WithContext(ctx))
			if rec.Code != tc.status || mailer.calls != tc.calls {
				t.Fatalf("status=%d, sends=%d; want %d, %d", rec.Code, mailer.calls, tc.status, tc.calls)
			}
			var result struct {
				ConfirmationSent bool `json:"confirmationSent"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}
			if result.ConfirmationSent != tc.sent {
				t.Fatalf("confirmationSent=%v, want %v", result.ConfirmationSent, tc.sent)
			}
		})
	}
}

func TestConfirmationTextSummarizesInput(t *testing.T) {
	req := validRequest()
	req.HouseNumber = "14"
	req.ParcelNote = "Flurstück 123/4\nEckgrundstück"
	text := confirmationText(buildLead(req))

	for _, want := range []string{
		"Guten Tag Thomas Ahrens",
		"werden jetzt von unserem Team geprüft",
		"E-Mail:      thomas@example.com",
		"Telefon:     +49 40 123456",
		"Grundstück:  Hauptstraße 14, 01067 Dresden",
		"Hinweis:     Flurstück 123/4\n             Eckgrundstück",
		"Antworten Sie einfach auf diese E-Mail",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("confirmation text missing %q:\n%s", want, text)
		}
	}
}

func TestConfirmationTextWithoutOptionalFields(t *testing.T) {
	text := confirmationText(buildLead(validRequest())) // no house number, no note
	if !strings.Contains(text, "Grundstück:  Hauptstraße, 01067 Dresden") {
		t.Errorf("address without house number rendered wrong:\n%s", text)
	}
	if strings.Contains(text, "Hinweis:") {
		t.Error("empty note must not be shown")
	}
}
