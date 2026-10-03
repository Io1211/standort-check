package backend

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

// handleCreateLead is the public endpoint behind the form.
//
// The response deliberately does not say whether the lead was a duplicate or
// return its ID: a public endpoint must not reveal whether an email address
// or address already exists in our system.
func (s *Server) handleCreateLead(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)

	var req CreateLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	_, err := s.leads.Create(r.Context(), req)
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
		writeJSON(w, http.StatusCreated, map[string]string{"status": "received"})
	}
}
