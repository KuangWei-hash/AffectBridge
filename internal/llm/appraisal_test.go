package llm

import (
	"context"
	"encoding/json"
	"testing"
)

type appraisalStub struct {
	opts options
}

func (s *appraisalStub) Complete(_ context.Context, _ string, opts ...Option) (string, error) {
	for _, opt := range opts {
		opt(&s.opts)
	}
	return `{"agency":"other","desirability":0.8,"unexpectedness":0.4,"blameworthiness":0,"praiseworthiness":0.9}`, nil
}

func TestAppraiseRequestsJSONSchema(t *testing.T) {
	stub := &appraisalStub{}
	got, err := Appraise(context.Background(), stub, "received a gift")
	if err != nil {
		t.Fatalf("Appraise() error = %v", err)
	}
	if got.Agency != "other" || got.Desirability != 0.8 {
		t.Fatalf("Appraise() = %+v", got)
	}
	if len(stub.opts.jsonSchema) == 0 {
		t.Fatal("Appraise() did not request JSON schema output")
	}
	if stub.opts.maxTokens != 512 {
		t.Fatalf("maxTokens = %d, want 512", stub.opts.maxTokens)
	}
	var schema map[string]any
	if err := json.Unmarshal(stub.opts.jsonSchema, &schema); err != nil {
		t.Fatalf("JSON schema is invalid: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("schema type = %v, want object", schema["type"])
	}
}
