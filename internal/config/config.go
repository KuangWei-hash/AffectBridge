// Package config centralizes runtime configuration.
//
// Configuration is read from environment variables. Defaults are
// chosen so the server can boot without any setup for development.
package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// HTTP listen port.
	Port string

	// Path to the locally installed ALMA runtime (informational).
	// AffectBridge does not read this directly; it is passed to the
	// ALMA backend if/when the runtime exposes its own HTTP server.
	ALMAHome string

	// HTTP address of a running ALMA runtime. When ALMAHome and
	// ALMAAddr are both set, the ALMA affect engine is wired in.
	// Otherwise a no-op engine is used.
	ALMAAddr string

	// LLM provider identifier. "openai", "anthropic", "local", etc.
	LLMProvider string

	// API key for the chosen LLM provider. Required for real calls.
	LLMAPIKey string

	// Model name for the chosen LLM provider.
	LLMModel string

	// Optional base URL override. Useful for OpenAI-compatible
	// proxies, local gateways, or self-hosted providers.
	LLMBaseURL string

	// LLMMaxConcurrent caps in-flight LLM calls per provider. When
	// the cap is reached the limiter returns ErrBusy instead of
	// queueing. <= 0 disables the limit.
	LLMMaxConcurrent int

	// LLMFallback is an ordered list of additional provider names to
	// try when the primary provider returns an error (including
	// ErrBusy). Empty = no fallback. All providers in the chain
	// share the same API key, model, and base URL; for heterogeneous
	// setups, extend the config.
	LLMFallback []string
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		ALMAHome:         os.Getenv("ALMA_HOME"),
		ALMAAddr:         getEnv("ALMA_ADDR", "http://localhost:9090"),
		LLMProvider:      getEnv("LLM_PROVIDER", "openai"),
		LLMAPIKey:        os.Getenv("LLM_API_KEY"),
		LLMModel:         getEnv("LLM_MODEL", "gpt-4o-mini"),
		LLMBaseURL:       os.Getenv("LLM_BASE_URL"),
		LLMMaxConcurrent: getEnvInt("LLM_MAX_CONCURRENT", 32),
		LLMFallback:      parseCSV(os.Getenv("LLM_FALLBACK")),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
