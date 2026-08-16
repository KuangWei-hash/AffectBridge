package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileBuildsALMAAddress(t *testing.T) {
	path := writeConfig(t, `{
  "server": {"port": 8080},
  "alma": {"host": "127.0.0.1", "port": 8081},
  "llm": {"provider": "openai", "host": "127.0.0.1", "port": 1234, "model": "deepseek/deepseek-r1-0528-qwen3-8b", "max_concurrent": 1}
}`)

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.ALMAAddr != "http://127.0.0.1:8081" {
		t.Fatalf("ALMAAddr = %q, want %q", cfg.ALMAAddr, "http://127.0.0.1:8081")
	}
	if cfg.LLMBaseURL != "http://127.0.0.1:1234/v1" {
		t.Fatalf("LLMBaseURL = %q, want %q", cfg.LLMBaseURL, "http://127.0.0.1:1234/v1")
	}
	if cfg.LLMModel != "deepseek/deepseek-r1-0528-qwen3-8b" {
		t.Fatalf("LLMModel = %q, want %q", cfg.LLMModel, "deepseek/deepseek-r1-0528-qwen3-8b")
	}
}

func TestLoadFileSupportsHostname(t *testing.T) {
	path := writeConfig(t, `{
  "server": {"port": 8080},
  "alma": {"host": "localhost", "port": 8081},
  "llm": {"provider": "openai", "host": "localhost", "port": 1234, "model": "deepseek/deepseek-r1-0528-qwen3-8b", "max_concurrent": 1}
}`)

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if cfg.ALMAAddr != "http://localhost:8081" {
		t.Fatalf("ALMAAddr = %q, want %q", cfg.ALMAAddr, "http://localhost:8081")
	}
}

func TestLoadFileRejectsInvalidSettings(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "missing ALMA host",
			content: `{"server":{"port":8080},"alma":{"port":8081}}`,
			wantErr: "alma.host is required",
		},
		{
			name:    "host contains scheme",
			content: `{"server":{"port":8080},"alma":{"host":"http://localhost","port":8081}}`,
			wantErr: "hostname or IP address",
		},
		{
			name:    "invalid ALMA port",
			content: `{"server":{"port":8080},"alma":{"host":"localhost","port":0}}`,
			wantErr: "alma.port",
		},
		{
			name:    "invalid server port",
			content: `{"server":{"port":0},"alma":{"host":"localhost","port":8081}}`,
			wantErr: "server.port",
		},
		{
			name:    "missing LLM host",
			content: `{"server":{"port":8080},"alma":{"host":"localhost","port":8081},"llm":{"provider":"openai","port":1234,"model":"qwen3-8b","max_concurrent":1}}`,
			wantErr: "llm.host is required",
		},
		{
			name:    "missing LLM model",
			content: `{"server":{"port":8080},"alma":{"host":"localhost","port":8081},"llm":{"provider":"openai","host":"localhost","port":1234,"max_concurrent":1}}`,
			wantErr: "llm.model is required",
		},
		{
			name:    "invalid LLM concurrency",
			content: `{"server":{"port":8080},"alma":{"host":"localhost","port":8081},"llm":{"provider":"openai","host":"localhost","port":1234,"model":"qwen3-8b","max_concurrent":0}}`,
			wantErr: "llm.max_concurrent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFile(writeConfig(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("LoadFile() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
