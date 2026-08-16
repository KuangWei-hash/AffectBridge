// Package api wires the HTTP routes for AffectBridge.
//
// The router is built from the application configuration so the
// affect backend (ALMA vs noop) and the LLM provider can be swapped
// without touching the controllers.
package api

import (
	"log"
	"net/http"

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

	// ALMA is the affect engine. Its address is assembled from the
	// required alma.host and alma.port values in config.json.
	affectEngine := alma.NewEngine(cfg.ALMAAddr)

	// The LLM client routes through pgEdge's unified library. Local
	// OpenAI-compatible servers such as LM Studio do not need an API key.
	//
	// Wiring: pgEdge client -> Limiter.
	var llmClient llm.Client
	primary, err := llm.NewPgEdgeClient(cfg.LLMProvider, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMBaseURL)
	if err != nil {
		log.Printf("llm: pgEdge client init failed: %v; falling back to noop", err)
		llmClient = llm.NewNoopClient()
	} else {
		llmClient = llm.NewLimiter(primary, cfg.LLMMaxConcurrent)
		log.Printf("llm: provider=%s model=%s base_url=%s max_concurrent=%d",
			cfg.LLMProvider, cfg.LLMModel, cfg.LLMBaseURL, cfg.LLMMaxConcurrent)
	}

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
