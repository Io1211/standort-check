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
		Subject     string         `json:"subject"`
		TextContent string         `json:"textContent"`
	}{
		Sender:      emailContact{Email: m.from, Name: "Standort-Check"},
		To:          []emailContact{{Email: lead.Email, Name: lead.FirstName + " " + lead.LastName}},
		Subject:     "Bestätigung Ihrer Anfrage zum Standort-Check",
		TextContent: fmt.Sprintf("Guten Tag %s %s,\n\nvielen Dank für Ihre Anfrage zum kostenlosen Standort-Check. Wir haben Ihre Angaben erhalten.\n\nUnser Team prüft den Standort und meldet sich in Kürze telefonisch oder per E-Mail bei Ihnen. Sie müssen das Formular nicht erneut absenden.\n\nFreundliche Grüße\nIhr Standort-Check-Team", lead.FirstName, lead.LastName),
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
