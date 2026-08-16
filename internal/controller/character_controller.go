// Package controller is the HTTP layer of AffectBridge.
//
// Controllers parse requests, call services, and write JSON
// responses. They do not contain business logic.
package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
	"github.com/KuangWei-hash/AffectBridge/internal/service"
)

type CharacterController struct {
	svc *service.CharacterService
}

func NewCharacterController(svc *service.CharacterService) *CharacterController {
	return &CharacterController{svc: svc}
}

type createCharacterRequest struct {
	Name        string            `json:"name"`
	Personality model.Personality `json:"personality"`
}

func (c *CharacterController) Create(w http.ResponseWriter, r *http.Request) {
	var req createCharacterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errMissingName)
		return
	}

	ch, err := c.svc.Create(req.Name, req.Personality)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, ch)
}

func (c *CharacterController) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ch, err := c.svc.Get(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}
