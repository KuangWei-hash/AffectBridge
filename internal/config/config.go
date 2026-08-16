// Package config centralizes runtime configuration.
//
// Server, ALMA, and LLM connection settings are read from config.json.
// Only optional secrets remain environment variables.
package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const DefaultPath = "config.json"

type Config struct {
	// HTTP listen port.
	Port string

	// HTTP address assembled from alma.host and alma.port in config.json.
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

	// Optional OpenAI-compatible reasoning effort (for example "none").
	LLMReasoningEffort string
}

type fileConfig struct {
	Server struct {
		Port int `json:"port"`
	} `json:"server"`
	ALMA struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"alma"`
	LLM struct {
		Provider        string `json:"provider"`
		BaseURL         string `json:"base_url"`
		Model           string `json:"model"`
		MaxConcurrent   int    `json:"max_concurrent"`
		ReasoningEffort string `json:"reasoning_effort"`
	} `json:"llm"`
}

func Load() (*Config, error) {
	return LoadFile(DefaultPath)
}

func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var file fileConfig
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	file.ALMA.Host = strings.TrimSpace(file.ALMA.Host)
	file.LLM.Provider = strings.TrimSpace(file.LLM.Provider)
	file.LLM.BaseURL = strings.TrimSpace(file.LLM.BaseURL)
	file.LLM.Model = strings.TrimSpace(file.LLM.Model)
	file.LLM.ReasoningEffort = strings.TrimSpace(file.LLM.ReasoningEffort)
	if file.Server.Port < 1 || file.Server.Port > 65535 {
		return nil, fmt.Errorf("config: server.port must be between 1 and 65535")
	}
	if file.ALMA.Host == "" {
		return nil, fmt.Errorf("config: alma.host is required")
	}
	if strings.Contains(file.ALMA.Host, "://") {
		return nil, fmt.Errorf("config: alma.host must contain only a hostname or IP address")
	}
	if file.ALMA.Port < 1 || file.ALMA.Port > 65535 {
		return nil, fmt.Errorf("config: alma.port must be between 1 and 65535")
	}
	if file.LLM.Provider == "" {
		return nil, fmt.Errorf("config: llm.provider is required")
	}
	if file.LLM.BaseURL == "" {
		return nil, fmt.Errorf("config: llm.base_url is required")
	}
	parsedLLMURL, err := url.Parse(file.LLM.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("config: llm.base_url is invalid: %w", err)
	}
	if parsedLLMURL.Scheme != "http" && parsedLLMURL.Scheme != "https" {
		return nil, fmt.Errorf("config: llm.base_url scheme must be http or https")
	}
	if parsedLLMURL.Hostname() == "" {
		return nil, fmt.Errorf("config: llm.base_url must include a hostname")
	}
	if parsedLLMURL.User != nil || parsedLLMURL.RawQuery != "" || parsedLLMURL.Fragment != "" {
		return nil, fmt.Errorf("config: llm.base_url must not include credentials, query parameters, or a fragment")
	}
	if file.LLM.Model == "" {
		return nil, fmt.Errorf("config: llm.model is required")
	}
	if file.LLM.MaxConcurrent < 1 {
		return nil, fmt.Errorf("config: llm.max_concurrent must be at least 1")
	}
	if file.LLM.ReasoningEffort != "" {
		switch file.LLM.ReasoningEffort {
		case "none", "minimal", "low", "medium", "high":
		default:
			return nil, fmt.Errorf("config: llm.reasoning_effort must be none, minimal, low, medium, or high")
		}
	}

	almaAddr := "http://" + net.JoinHostPort(file.ALMA.Host, strconv.Itoa(file.ALMA.Port))
	llmBaseURL := strings.TrimRight(file.LLM.BaseURL, "/")

	return &Config{
		Port:               strconv.Itoa(file.Server.Port),
		ALMAAddr:           almaAddr,
		LLMProvider:        file.LLM.Provider,
		LLMAPIKey:          os.Getenv("LLM_API_KEY"),
		LLMModel:           file.LLM.Model,
		LLMBaseURL:         llmBaseURL,
		LLMMaxConcurrent:   file.LLM.MaxConcurrent,
		LLMReasoningEffort: file.LLM.ReasoningEffort,
	}, nil
}
