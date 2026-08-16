package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	pgedgellm "github.com/pgEdge/pgedge-go-llm-lib/llm"
	// Register all built-in providers (openai, anthropic, gemini,
	// ollama, voyage). Import specific subpackages instead to keep
	// the binary smaller.
	_ "github.com/pgEdge/pgedge-go-llm-lib/llm/all"
)

// pgedgeClient wraps pgEdge's unified LLM client so it satisfies our
// local Client interface. The pgEdge library handles provider-specific
// quirks (auth, retries, JSON mode, streaming) so the wrapper is
// deliberately thin.
type pgedgeClient struct {
	inner pgedgellm.Client
}

// NewPgEdgeClient creates a Client backed by pgEdge's library.
// provider is "openai", "anthropic", "gemini", "ollama", or any
// other pgEdge-registered provider. model and baseURL are forwarded;
// pass empty strings to use defaults.
func NewPgEdgeClient(provider, apiKey, model, baseURL, reasoningEffort string) (Client, error) {
	opts := pgedgellm.Options{
		APIKey: apiKey,
		Model:  model,
	}
	if baseURL != "" {
		opts.BaseURL = baseURL
	}
	if reasoningEffort != "" {
		opts.HTTPClient = &http.Client{Transport: &reasoningEffortTransport{
			base:   http.DefaultTransport,
			effort: reasoningEffort,
		}}
	}
	inner, err := pgedgellm.NewClient(provider, opts)
	if err != nil {
		return nil, fmt.Errorf("pgedge: new client: %w", err)
	}
	return &pgedgeClient{inner: inner}, nil
}

// reasoningEffortTransport fills a gap in pgEdge v0.3.0, whose unified
// ChatRequest does not yet expose OpenAI-compatible reasoning_effort. It only
// touches chat-completion JSON requests and otherwise passes requests through.
type reasoningEffortTransport struct {
	base   http.RoundTripper
	effort string
}

func (t *reasoningEffortTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodPost || !strings.HasSuffix(req.URL.Path, "/chat/completions") || req.Body == nil {
		return t.base.RoundTrip(req)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	payload["reasoning_effort"] = t.effort
	body, err = json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	clone := req.Clone(req.Context())
	clone.Body = io.NopCloser(bytes.NewReader(body))
	clone.ContentLength = int64(len(body))
	clone.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	return t.base.RoundTrip(clone)
}

// Providers returns the list of providers registered with pgEdge.
// Useful for config validation and /healthz output.
func Providers() []string {
	return pgedgellm.RegisteredProviders()
}

func (c *pgedgeClient) Complete(ctx context.Context, prompt string, opts ...Option) (string, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	var messages []pgedgellm.Message
	if o.system != "" {
		messages = append(messages, pgedgellm.SystemText(o.system))
	}
	messages = append(messages, pgedgellm.UserText(prompt))

	req := pgedgellm.ChatRequest{Messages: messages}
	if o.temperature != 0 {
		t := o.temperature
		req.Temperature = &t
	}
	if o.maxTokens > 0 {
		n := o.maxTokens
		req.MaxTokens = &n
	}
	if len(o.jsonSchema) > 0 {
		req.ResponseFormat = &pgedgellm.ResponseFormat{
			Type:       pgedgellm.ResponseFormatJSONSchema,
			JSONSchema: o.jsonSchema,
		}
	} else if o.jsonMode {
		req.ResponseFormat = &pgedgellm.ResponseFormat{Type: pgedgellm.ResponseFormatJSON}
	}

	resp, err := c.inner.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	for _, block := range resp.Content {
		if block.Type == pgedgellm.BlockText {
			return block.Text, nil
		}
	}
	return "", nil
}
