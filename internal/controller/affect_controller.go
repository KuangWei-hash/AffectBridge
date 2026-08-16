package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
	"github.com/KuangWei-hash/AffectBridge/internal/service"
)

type AffectController struct {
	charSvc *service.CharacterService
	affSvc  *service.AffectService
}

func NewAffectController(charSvc *service.CharacterService, affSvc *service.AffectService) *AffectController {
	return &AffectController{charSvc: charSvc, affSvc: affSvc}
}

// Snapshot returns the current affective state of a character.
func (c *AffectController) Snapshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ch, err := c.charSvc.Get(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	snap := c.affSvc.Snapshot(ch)
	writeJSON(w, http.StatusOK, snap)
}

// Apply feeds an externally supplied appraisal into the affect
// engine and persists the updated character. This endpoint is
// useful for tests and for clients that want to skip the LLM-based
// appraisal step.
func (c *AffectController) Apply(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ch, err := c.charSvc.Get(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	var appraisal model.Appraisal
	if err := json.NewDecoder(r.Body).Decode(&appraisal); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := c.affSvc.Apply(ch, appraisal)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
