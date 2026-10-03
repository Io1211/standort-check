package backend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"
)

// handleCreateLead is the public endpoint behind the form.
//
// The response deliberately does not say whether the lead was a duplicate or
// return its ID: a public endpoint must not reveal whether an email address
// or address already exists in our system.
func (s *Server) handleCreateLead(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage.")
		return
	}
	// encoding/json silently replaces invalid UTF-8 with U+FFFD, which would
	// store "Hauptstra�e". Reject such bodies (e.g. Latin-1 encoded) instead.
	if !utf8.Valid(body) {
		writeError(w, http.StatusBadRequest, "Ungültige Zeichenkodierung (UTF-8 erwartet).")
		return
	}

	var req CreateLeadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage.")
		return
	}

	// Honeypot filled → almost certainly a bot. Pretend success so the bot
	// learns nothing, but store nothing and send no email.
	if strings.TrimSpace(req.Website) != "" {
		log.Printf("honeypot triggered, submission dropped")
		writeJSON(w, http.StatusCreated, map[string]string{"status": "received"})
		return
	}

	lead, err := s.leads.Create(r.Context(), req)
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":  "Bitte prüfen Sie Ihre Eingaben.",
			"fields": verr.Fields,
		})
	case err != nil:
		log.Printf("create lead failed: %v", err)
		writeError(w, http.StatusInternalServerError,
			"Ihre Anfrage konnte gerade nicht gespeichert werden. Bitte versuchen Sie es in einigen Minuten erneut.")
	default:
		confirmationSent := false
		if s.mailer != nil {
			// Finish this bounded send before returning: a serverless runtime
			// can freeze background goroutines when the response is complete.
			// A browser disconnect must not cancel mail for an already saved lead.
			ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), confirmationTimeout)
			err := s.mailer.SendConfirmation(ctx, lead)
			cancel()
			if err != nil {
				log.Printf("confirmation email failed for lead %s: %v", lead.ID, err)
			} else {
				confirmationSent = true
			}
		} else {
			log.Printf("confirmation email disabled: BREVO_API_KEY and EMAIL_FROM must be configured")
		}
		writeJSON(w, http.StatusCreated, map[string]any{"status": "received", "confirmationSent": confirmationSent})
	}
}
