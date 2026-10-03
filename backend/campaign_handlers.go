package backend

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := s.repo.ListCampaigns(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaigns": campaigns})
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeCampaign(w, r)
	if !ok {
		return
	}
	c, err := s.repo.CreateCampaign(r.Context(), req)
	if !writeCampaignError(w, err) {
		writeJSON(w, http.StatusCreated, c)
	}
}

func (s *Server) handleUpdateCampaign(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validUUID(id) {
		writeError(w, http.StatusNotFound, "Kampagne nicht gefunden.")
		return
	}
	req, ok := decodeCampaign(w, r)
	if !ok {
		return
	}
	c, err := s.repo.UpdateCampaign(r.Context(), id, req)
	if !writeCampaignError(w, err) {
		writeJSON(w, http.StatusOK, c)
	}
}

func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validUUID(id) {
		writeError(w, http.StatusNotFound, "Kampagne nicht gefunden.")
		return
	}
	if !writeCampaignError(w, s.repo.DeleteCampaign(r.Context(), id)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func decodeCampaign(w http.ResponseWriter, r *http.Request) (CampaignRequest, bool) {
	var req CampaignRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Ungültige Anfrage.")
		return req, false
	}
	req = req.normalize()
	if verr := req.validate(); verr != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "Bitte Eingaben prüfen.", "fields": verr.Fields})
		return req, false
	}
	return req, true
}

// writeCampaignError writes the response for err and reports whether it did.
func writeCampaignError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "Kampagne nicht gefunden.")
	case errors.Is(err, ErrDuplicateCampaign):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":  "Einen Link mit genau diesen Werten gibt es schon.",
			"fields": map[string]string{"utmCampaign": "Diese Kombination aus Quelle, Medium, Kampagne und Anzeige existiert bereits."},
		})
	default:
		internalError(w, err)
	}
	return true
}
