package llm

import "testing"

func TestNewPgEdgeClientAllowsLocalServerWithoutAPIKey(t *testing.T) {
	client, err := NewPgEdgeClient(
		"openai",
		"",
		"deepseek/deepseek-r1-0528-qwen3-8b",
		"http://127.0.0.1:1234/v1",
	)
	if err != nil {
		t.Fatalf("NewPgEdgeClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewPgEdgeClient() returned nil client")
	}
}
