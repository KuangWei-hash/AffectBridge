// Package llm abstracts the language model provider.
//
// The first concrete provider can be OpenAI, Anthropic, or a local
// model. Other parts of AffectBridge depend only on the Client
// interface, so the provider can be swapped by changing one wire.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// ErrBusy is returned by the limiter when the configured concurrency
// cap is reached. The caller should NOT retry; either fail fast or
// route to a different provider.
var ErrBusy = errors.New("llm: at capacity")

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
	jsonSchema  json.RawMessage
}

func WithTemperature(t float64) Option { return func(o *options) { o.temperature = t } }
func WithMaxTokens(n int) Option       { return func(o *options) { o.maxTokens = n } }
func WithSystem(s string) Option       { return func(o *options) { o.system = s } }

// WithJSONMode instructs the provider to return a JSON object. The
// Interpreter (llm.Appraise) uses it. Providers that do not support
// JSON mode ignore it.
func WithJSONMode() Option { return func(o *options) { o.jsonMode = true } }

// WithJSONSchema instructs the provider to return an object matching schema.
// OpenAI-compatible servers that reject the legacy json_object format, such as
// recent LM Studio versions, can use this structured-output mode instead.
func WithJSONSchema(schema string) Option {
	return func(o *options) { o.jsonSchema = json.RawMessage(schema) }
}

// NewNoopClient returns a client that always returns a placeholder
// response. It lets the rest of the system boot without an LLM
// configured, which is useful for development and tests.
func NewNoopClient() Client {
	return &noopClient{}
}

type noopClient struct{}

func (n *noopClient) Complete(_ context.Context, prompt string, opts ...Option) (string, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	if len(o.jsonSchema) > 0 && strings.Contains(string(o.jsonSchema), `"new_topic_event"`) {
		var input struct {
			CharacterName string `json:"character_name"`
		}
		if err := json.Unmarshal([]byte(prompt), &input); err == nil && input.CharacterName != "" {
			result, _ := json.Marshal(map[string]any{
				"decision":      "new_topic_event",
				"relation":      "new",
				"topic_id":      "",
				"event_id":      "",
				"topic_summary": "未設定 LLM 的故事脈絡",
				"event_summary": "未設定 LLM 的故事事件",
				"event_type":    "unknown",
				"subject":       input.CharacterName,
				"target":        "未明對象",
				"status":        "active",
				"reason":        "未設定 LLM，因此建立獨立事件。",
			})
			return string(result), nil
		}
	}
	if len(o.jsonSchema) > 0 && strings.Contains(string(o.jsonSchema), `"matched"`) {
		var input struct {
			TargetTag string `json:"target_tag"`
		}
		if err := json.Unmarshal([]byte(prompt), &input); err == nil && input.TargetTag != "" {
			result, _ := json.Marshal(map[string]any{
				"tag":       input.TargetTag,
				"matched":   false,
				"reason":    "未設定 LLM，沒有可判斷的證據。",
				"intensity": "不適用",
			})
			return string(result), nil
		}
	}
	if o.jsonMode || len(o.jsonSchema) > 0 {
		// Return a valid zero Appraisal so the Interpreter pipeline
		// can still run without a real LLM configured.
		return `{"agency":"none","desirability":0,"unexpectedness":0,"blameworthiness":0,"praiseworthiness":0}`, nil
	}
	return "[no llm configured]", nil
}
