// Package api wires the HTTP routes for AffectBridge.
//
// The router is built from the application configuration so the
// affect backend (ALMA vs noop) and the LLM provider can be swapped
// without touching the controllers.
package api

import (
	"log"
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

	// The LLM client. When LLM_API_KEY is set, route through pgEdge's
	// unified library (OpenAI / Anthropic / Gemini / Ollama).
	// Otherwise fall back to a no-op client so the server can boot.
	//
	// Wiring: primary pgEdge client -> Limiter -> [optional Router
	// over fallbacks]. Each fallback gets its own Limiter so capacity
	// is tracked per provider.
	var llmClient llm.Client
	if cfg.LLMAPIKey != "" {
		primary, err := llm.NewPgEdgeClient(cfg.LLMProvider, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMBaseURL)
		if err != nil {
			log.Printf("llm: pgEdge client init failed: %v; falling back to noop", err)
			llmClient = llm.NewNoopClient()
		} else {
			providers := []llm.Client{llm.NewLimiter(primary, cfg.LLMMaxConcurrent)}
			for _, fbProvider := range cfg.LLMFallback {
				fb, fbErr := llm.NewPgEdgeClient(fbProvider, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMBaseURL)
				if fbErr != nil {
					log.Printf("llm: fallback provider=%s init failed: %v", fbProvider, fbErr)
					continue
				}
				providers = append(providers, llm.NewLimiter(fb, cfg.LLMMaxConcurrent))
			}
			if len(providers) > 1 {
				llmClient = llm.NewRouter(providers...)
			} else {
				llmClient = providers[0]
			}
			log.Printf("llm: provider=%s model=%s max_concurrent=%d fallbacks=%v",
				cfg.LLMProvider, cfg.LLMModel, cfg.LLMMaxConcurrent, cfg.LLMFallback)
		}
	} else {
		llmClient = llm.NewNoopClient()
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
