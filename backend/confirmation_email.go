package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// confirmationTimeout bounds the whole send, including the one-off sender
// check on a cold start (two short Brevo calls).
const confirmationTimeout = 5 * time.Second

// senderCheckTTL is how long a positive sender check is trusted. A sender
// that disappears from Brevo is noticed within this window at the latest.
const senderCheckTTL = time.Hour

type confirmationSender interface {
	SendConfirmation(context.Context, Lead) error
}

type brevoMailer struct {
	apiKey          string
	from            string
	endpoint        string
	sendersEndpoint string
	client          *http.Client

	mu         sync.Mutex
	verifiedAt time.Time // zero until the sender has been found active in Brevo
}

func newConfirmationSender(cfg Config) confirmationSender {
	if strings.TrimSpace(cfg.BrevoAPIKey) == "" || strings.TrimSpace(cfg.EmailFrom) == "" {
		return nil
	}
	return &brevoMailer{
		apiKey:          strings.TrimSpace(cfg.BrevoAPIKey),
		from:            strings.TrimSpace(cfg.EmailFrom),
		endpoint:        "https://api.brevo.com/v3/smtp/email",
		sendersEndpoint: "https://api.brevo.com/v3/senders",
		client:          &http.Client{Timeout: confirmationTimeout},
	}
}

// ensureVerifiedSender guards against Brevo's asynchronous rejection: the
// send endpoint answers HTTP 201 even for a sender that is not verified in
// the account and only discards the mail afterwards ("Sending has been
// rejected because the sender you used ... is not valid"). Without this
// check the form would tell the customer that a confirmation was sent when
// nothing ever leaves Brevo. The sender list is fetched at most once per
// senderCheckTTL per process.
func (m *brevoMailer) ensureVerifiedSender(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.verifiedAt.IsZero() && time.Since(m.verifiedAt) < senderCheckTTL {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.sendersEndpoint, nil)
	if err != nil {
		return fmt.Errorf("prepare sender check: %w", err)
	}
	req.Header.Set("api-key", m.apiKey)
	req.Header.Set("Accept", "application/json")
	res, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("sender check: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 8<<10))
		return fmt.Errorf("sender check rejected by Brevo (HTTP %d)", res.StatusCode)
	}
	var list struct {
		Senders []struct {
			Email  string `json:"email"`
			Active bool   `json:"active"`
		} `json:"senders"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&list); err != nil {
		return fmt.Errorf("decode sender list: %w", err)
	}
	for _, s := range list.Senders {
		if strings.EqualFold(strings.TrimSpace(s.Email), m.from) {
			if !s.Active {
				return fmt.Errorf("sender %s exists in Brevo but is not verified yet; Brevo would discard the mail", m.from)
			}
			m.verifiedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("sender %s is not a verified Brevo sender (EMAIL_FROM must match a verified entry under Senders); Brevo would discard the mail", m.from)
}

type emailContact struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (m *brevoMailer) SendConfirmation(ctx context.Context, lead Lead) error {
	if err := m.ensureVerifiedSender(ctx); err != nil {
		return err
	}
	payload := struct {
		Sender      emailContact   `json:"sender"`
		To          []emailContact `json:"to"`
		ReplyTo     emailContact   `json:"replyTo"`
		Subject     string         `json:"subject"`
		TextContent string         `json:"textContent"`
	}{
		Sender: emailContact{Email: m.from, Name: "Standort-Check"},
		To:     []emailContact{{Email: lead.Email, Name: lead.FirstName + " " + lead.LastName}},
		// Without an own domain Brevo rewrites the sender to a technical
		// brevosend.com address; replies must still reach the team.
		ReplyTo:     emailContact{Email: m.from, Name: "Standort-Check"},
		Subject:     "Bestätigung Ihrer Anfrage zum Standort-Check",
		TextContent: confirmationText(lead),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode confirmation: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("prepare confirmation: %w", err)
	}
	req.Header.Set("api-key", m.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("send confirmation: %w", err)
	}
	defer res.Body.Close()
	// Drain a bounded body for connection reuse. Never log provider response
	// bodies: they can contain the recipient's personal data.
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 8<<10))
	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("confirmation rejected by Brevo (HTTP %d)", res.StatusCode)
	}
	return nil
}

// confirmationText is the plain-text confirmation. It repeats what the
// customer entered, so typos (e.g. a wrong postcode or phone number) can be
// spotted and corrected by simply replying. Plain text only: user input is
// never interpreted as HTML, and text mails are less likely to be flagged
// as spam.
func confirmationText(l Lead) string {
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }

	line("Guten Tag %s %s,", l.FirstName, l.LastName)
	line("")
	line("vielen Dank für Ihre Anfrage zum kostenlosen Standort-Check. Ihre Angaben sind bei uns eingegangen und werden jetzt von unserem Team geprüft.")
	line("")
	line("Ihre Angaben im Überblick")
	line("-------------------------")
	line("Name:        %s %s", l.FirstName, l.LastName)
	line("E-Mail:      %s", l.Email)
	line("Telefon:     %s", l.Phone)
	address := l.Street
	if l.HouseNumber != "" {
		address += " " + l.HouseNumber
	}
	line("Grundstück:  %s, %s %s", address, l.PostalCode, l.City)
	if l.ParcelNote != "" {
		line("Hinweis:     %s", strings.ReplaceAll(l.ParcelNote, "\n", "\n             "))
	}
	line("")
	line("Wie geht es weiter?")
	line("1. Wir prüfen Ihre Angaben und den Standort Ihres Grundstücks.")
	line("2. Eine Ansprechperson aus unserem Team meldet sich in Kürze telefonisch oder per E-Mail bei Ihnen.")
	line("")
	line("Stimmt etwas nicht? Antworten Sie einfach auf diese E-Mail. Sie müssen das Formular nicht erneut absenden.")
	line("")
	line("Freundliche Grüße")
	line("Ihr Standort-Check-Team")
	return b.String()
}
