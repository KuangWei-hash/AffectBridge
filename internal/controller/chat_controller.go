package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KuangWei-hash/AffectBridge/internal/repository"
	"github.com/KuangWei-hash/AffectBridge/internal/service"
)

type ChatController struct {
	charSvc *service.CharacterService
	chatSvc *service.ChatService
}

func NewChatController(charSvc *service.CharacterService, chatSvc *service.ChatService) *ChatController {
	return &ChatController{charSvc: charSvc, chatSvc: chatSvc}
}

type chatRequest struct {
	Message string `json:"message"`
}

// Send runs the full AffectBridge pipeline for a single player
// message and returns the character's reply together with the
// appraisal and updated affective state.
func (c *ChatController) Send(w http.ResponseWriter, r *http.Request) {
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

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, errEmptyMessage)
		return
	}

	reply, err := c.chatSvc.Send(r.Context(), ch, req.Message)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, reply)
}
