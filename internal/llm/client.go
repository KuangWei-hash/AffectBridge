// Package llm abstracts the language model provider.
//
// The first concrete provider can be OpenAI, Anthropic, or a local
// model. Other parts of AffectBridge depend only on the Client
// interface, so the provider can be swapped by changing one wire.
package llm

import "context"

// Client is a provider-agnostic LLM client.
type Client interface {
	Complete(ctx context.Context, prompt string, opts ...Option) (string, error)
}

// Option mutates a single completion request.
type Option func(*options)

type options struct {
	temperature float64
	maxTokens   int
	system      string
	jsonMode    bool
}

func WithTemperature(t float64) Option { return func(o *options) { o.temperature = t } }
func WithMaxTokens(n int) Option      { return func(o *options) { o.maxTokens = n } }
func WithSystem(s string) Option      { return func(o *options) { o.system = s } }
// WithJSONMode instructs the provider to return a JSON object. The
// Interpreter (llm.Appraise) uses it. Providers that do not support
// JSON mode ignore it.
func WithJSONMode() Option { return func(o *options) { o.jsonMode = true } }

// NewNoopClient returns a client that always returns a placeholder
// response. It lets the rest of the system boot without an LLM
// configured, which is useful for development and tests.
func NewNoopClient() Client {
	return &noopClient{}
}

type noopClient struct{}

func (n *noopClient) Complete(_ context.Context, _ string, opts ...Option) (string, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.jsonMode {
		// Return a valid zero Appraisal so the Interpreter pipeline
		// can still run without a real LLM configured.
		return `{"agency":"none","desirability":0,"unexpectedness":0,"blameworthiness":0,"praiseworthiness":0}`, nil
	}
	return "[no llm configured]", nil
}
