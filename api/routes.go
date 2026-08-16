// Package api wires the HTTP routes for AffectBridge.
//
// The router is built from the application configuration so the
// affect backend (ALMA vs noop) and the LLM provider can be swapped
// without touching the controllers.
package api

import (
	"net/http"

	"github.com/KuangWei-hash/AffectBridge/internal/affect"
	"github.com/KuangWei-hash/AffectBridge/internal/affect/alma"
	"github.com/KuangWei-hash/AffectBridge/internal/config"
	"github.com/KuangWei-hash/AffectBridge/internal/controller"
	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
	"github.com/KuangWei-hash/AffectBridge/internal/service"
)

func NewRouter(cfg *config.Config) *http.ServeMux {
	mux := http.NewServeMux()

	repo := repository.NewInMemoryCharacterRepository()

	// The affect engine. ALMA is the targeted backend; noop is used
	// until ALMA is wired up.
	var affectEngine affect.Engine
	if cfg.ALMAHome != "" && cfg.ALMAAddr != "" {
		affectEngine = alma.NewEngine(cfg.ALMAAddr)
	} else {
		affectEngine = affect.NewNoopEngine()
	}

	// The LLM client. Replace with a real provider when ready.
	llmClient := llm.NewNoopClient()

	charSvc := service.NewCharacterService(repo)
	affectSvc := service.NewAffectService(affectEngine, repo)
	appraisalSvc := service.NewAppraisalService(llmClient)
	chatSvc := service.NewChatService(llmClient, appraisalSvc, affectSvc)

	charCtrl := controller.NewCharacterController(charSvc)
	affectCtrl := controller.NewAffectController(charSvc, affectSvc)
	chatCtrl := controller.NewChatController(charSvc, chatSvc)

	// Health
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Characters
	mux.HandleFunc("POST /characters", charCtrl.Create)
	mux.HandleFunc("GET /characters/{id}", charCtrl.Get)

	// Affect
	mux.HandleFunc("GET /characters/{id}/affect", affectCtrl.Snapshot)
	mux.HandleFunc("POST /characters/{id}/affect", affectCtrl.Apply)

	// Chat
	mux.HandleFunc("POST /characters/{id}/chat", chatCtrl.Send)

	return mux
}
