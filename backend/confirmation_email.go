package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const confirmationTimeout = 3 * time.Second

type confirmationSender interface {
	SendConfirmation(context.Context, Lead) error
}

type brevoMailer struct {
	apiKey   string
	from     string
	endpoint string
	client   *http.Client
}

func newConfirmationSender(cfg Config) confirmationSender {
	if strings.TrimSpace(cfg.BrevoAPIKey) == "" || strings.TrimSpace(cfg.EmailFrom) == "" {
		return nil
	}
	return &brevoMailer{
		apiKey:   strings.TrimSpace(cfg.BrevoAPIKey),
		from:     strings.TrimSpace(cfg.EmailFrom),
		endpoint: "https://api.brevo.com/v3/smtp/email",
		client:   &http.Client{Timeout: confirmationTimeout},
	}
}

type emailContact struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (m *brevoMailer) SendConfirmation(ctx context.Context, lead Lead) error {
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
