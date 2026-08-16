package llm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestNewPgEdgeClientAllowsLocalServerWithoutAPIKey(t *testing.T) {
	client, err := NewPgEdgeClient(
		"openai",
		"",
		"deepseek/deepseek-r1-0528-qwen3-8b",
		"http://127.0.0.1:1234/v1",
		"",
	)
	if err != nil {
		t.Fatalf("NewPgEdgeClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewPgEdgeClient() returned nil client")
	}
}

func TestPgEdgeClientAddsReasoningEffort(t *testing.T) {
	base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if got := body["reasoning_effort"]; got != "none" {
			t.Fatalf("reasoning_effort = %v, want none", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(nil)),
			Header:     make(http.Header),
		}, nil
	})
	transport := &reasoningEffortTransport{base: base, effort: "none"}
	req, err := http.NewRequest(http.MethodPost, "http://localhost/v1/chat/completions",
		bytes.NewBufferString(`{"model":"qwen3.5-2b","messages":[]}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	_ = resp.Body.Close()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
