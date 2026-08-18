package llm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

type appraisalWorkerStub struct {
	prompt string
	opts   options
	result string
	err    error
}

func (s *appraisalWorkerStub) Complete(_ context.Context, prompt string, opts ...Option) (string, error) {
	s.prompt = prompt
	for _, opt := range opts {
		opt(&s.opts)
	}
	return s.result, s.err
}

func TestRunAppraisalWorkerUsesVariantAFullOntology(t *testing.T) {
	if AppraisalWorkerMaxOutputTokens != 128 {
		t.Fatalf("AppraisalWorkerMaxOutputTokens = %d, want 128", AppraisalWorkerMaxOutputTokens)
	}
	stub := &appraisalWorkerStub{
		result: `{"tag":"GoodEventForBadOther","matched":true,"reason":"敵人取得重要資源。","intensity":"強烈"}`,
	}
	got, err := RunAppraisalWorker(
		context.Background(),
		stub,
		model.GoodEventForBadOther,
		"Lisa",
		"Lisa 對敵人界線鮮明。",
		"William 取得資源；Lisa 敵視 William。",
	)
	if err != nil {
		t.Fatalf("RunAppraisalWorker() error = %v", err)
	}
	if got.Tag != model.GoodEventForBadOther || !got.Matched || got.Intensity != model.IntensityStrong {
		t.Fatalf("result = %+v", got)
	}
	if stub.opts.maxTokens != AppraisalWorkerMaxOutputTokens {
		t.Fatalf("max tokens = %d, want %d", stub.opts.maxTokens, AppraisalWorkerMaxOutputTokens)
	}
	if len(stub.opts.jsonSchema) == 0 || !strings.Contains(string(stub.opts.jsonSchema), `"GoodEventForBadOther"`) {
		t.Fatal("worker did not request a target-specific JSON schema")
	}
	for _, definition := range appraisalWorkerDefinitions {
		question := specialize(definition.question, "Lisa")
		if !strings.Contains(stub.opts.system, question) {
			t.Fatalf("system prompt missing ontology question for %s", definition.tag)
		}
	}
	if !strings.Contains(stub.opts.system, "問題 4（GoodEventForBadOther）") {
		t.Fatal("system prompt does not designate only question 4")
	}

	var input appraisalWorkerInput
	if err := json.Unmarshal([]byte(stub.prompt), &input); err != nil {
		t.Fatalf("prompt is not JSON: %v", err)
	}
	if input.TargetTag != string(model.GoodEventForBadOther) || input.CharacterName != "Lisa" {
		t.Fatalf("input = %+v", input)
	}
	if input.CharacterView != "Lisa 對敵人界線鮮明。" || input.Story != "William 取得資源；Lisa 敵視 William。" {
		t.Fatalf("input data changed: %+v", input)
	}
}

func TestAppraisalWorkerDefinitionsMatchCanonicalTags(t *testing.T) {
	for i, tag := range model.AllAppraisalTags {
		if appraisalWorkerDefinitions[i].tag != tag {
			t.Fatalf("definition[%d].tag = %s, want %s", i, appraisalWorkerDefinitions[i].tag, tag)
		}
		if appraisalWorkerNumber(tag) != i+1 {
			t.Fatalf("worker number for %s = %d, want %d", tag, appraisalWorkerNumber(tag), i+1)
		}
	}
}

func TestRunAppraisalWorkerSuppliesMissingCharacterView(t *testing.T) {
	stub := &appraisalWorkerStub{
		result: `{"tag":"GoodEvent","matched":false,"reason":"沒有有利事件。","intensity":"不適用"}`,
	}
	_, err := RunAppraisalWorker(context.Background(), stub, model.GoodEvent, "Lisa", "  ", "Lisa 等待玩家。")
	if err != nil {
		t.Fatalf("RunAppraisalWorker() error = %v", err)
	}
	var input appraisalWorkerInput
	if err := json.Unmarshal([]byte(stub.prompt), &input); err != nil {
		t.Fatal(err)
	}
	if input.CharacterView != "（未提供角色特殊看法）" {
		t.Fatalf("CharacterView = %q", input.CharacterView)
	}
}

func TestRunAppraisalWorkerRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		tag       model.AppraisalTag
		character string
		story     string
		want      string
	}{
		{name: "empty character", tag: model.GoodEvent, story: "故事", want: ErrEmptyCharacterName.Error()},
		{name: "empty story", tag: model.GoodEvent, character: "Lisa", want: ErrEmptyAppraisalStory.Error()},
		{name: "invalid tag", tag: "Unknown", character: "Lisa", story: "故事", want: "invalid tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RunAppraisalWorker(context.Background(), &appraisalWorkerStub{}, tt.tag, tt.character, "", tt.story)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRunAppraisalWorkerValidatesOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "empty", output: "  ", want: ErrEmptyWorkerOutput.Error()},
		{name: "invalid json", output: `{`, want: "parse output"},
		{name: "wrong tag", output: `{"tag":"BadEvent","matched":false,"reason":"理由","intensity":"不適用"}`, want: "want \"GoodEvent\""},
		{name: "empty reason", output: `{"tag":"GoodEvent","matched":false,"reason":" ","intensity":"不適用"}`, want: "empty reason"},
		{name: "matched without intensity", output: `{"tag":"GoodEvent","matched":true,"reason":"理由","intensity":"不適用"}`, want: "matched but intensity"},
		{name: "unmatched with intensity", output: `{"tag":"GoodEvent","matched":false,"reason":"理由","intensity":"中等"}`, want: "did not match"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &appraisalWorkerStub{result: tt.output}
			_, err := RunAppraisalWorker(context.Background(), stub, model.GoodEvent, "Lisa", "", "故事")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRunAppraisalWorkerWrapsProviderError(t *testing.T) {
	want := errors.New("provider unavailable")
	stub := &appraisalWorkerStub{err: want}
	_, err := RunAppraisalWorker(context.Background(), stub, model.GoodEvent, "Lisa", "", "故事")
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want wrapping %v", err, want)
	}
}

func TestRunAppraisalWorkerSupportsNoopClient(t *testing.T) {
	got, err := RunAppraisalWorker(context.Background(), NewNoopClient(), model.NastyThing, "Lisa", "", "故事")
	if err != nil {
		t.Fatalf("RunAppraisalWorker() error = %v", err)
	}
	if got.Tag != model.NastyThing || got.Matched || got.Intensity != model.IntensityNotApplicable {
		t.Fatalf("result = %+v", got)
	}
}
