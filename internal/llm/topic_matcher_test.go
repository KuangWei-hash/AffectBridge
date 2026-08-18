package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

type topicMatcherStub struct {
	prompt string
	opts   options
	result string
	err    error
}

func (s *topicMatcherStub) Complete(_ context.Context, prompt string, opts ...Option) (string, error) {
	s.prompt = prompt
	for _, opt := range opts {
		opt(&s.opts)
	}
	return s.result, s.err
}

func TestMatchTopicEventReusesPendingCandidateForConfirmation(t *testing.T) {
	topics, events := matcherCandidates(model.EventStatusPending)
	stub := &topicMatcherStub{result: `{
      "decision":"reuse_event","relation":"confirmation",
      "topic_id":"T-000001","event_id":"E-000001",
      "topic_summary":"","event_summary":"","event_type":"",
      "subject":"","target":"","status":"",
      "reason":"玩家已履行先前的返回承諾。"
    }`}
	got, err := MatchTopicEvent(context.Background(), stub, "Lisa", "玩家返回 Lisa 身邊。", topics, events)
	if err != nil {
		t.Fatalf("MatchTopicEvent() error = %v", err)
	}
	if got.EventID != "E-000001" || got.Relation != model.EventRelationConfirmation {
		t.Fatalf("decision = %+v", got)
	}
	if stub.opts.maxTokens != TopicMatcherMaxOutputTokens || TopicMatcherMaxOutputTokens != 128 {
		t.Fatalf("max tokens = %d, want 128", stub.opts.maxTokens)
	}
	if stub.opts.system != topicMatcherSystemPrompt || len(stub.opts.jsonSchema) == 0 {
		t.Fatal("matcher did not request the canonical system prompt and JSON schema")
	}
	var input topicMatcherInput
	if err := json.Unmarshal([]byte(stub.prompt), &input); err != nil {
		t.Fatalf("prompt is not JSON: %v", err)
	}
	if len(input.Events) != 1 || input.Events[0].ID != "E-000001" || input.Story != "玩家返回 Lisa 身邊。" {
		t.Fatalf("input = %+v", input)
	}
}

func TestMatchTopicEventCreatesNewTopicWhenNoCandidates(t *testing.T) {
	stub := &topicMatcherStub{result: `{
      "decision":"new_topic_event","relation":"new",
      "topic_id":"","event_id":"",
      "topic_summary":"玩家與 Lisa 的承諾",
      "event_summary":"玩家承諾返回 Lisa 身邊","event_type":"promise",
      "subject":"玩家","target":"Lisa","status":"pending",
      "reason":"候選清單中沒有相關事件。"
    }`}
	got, err := MatchTopicEvent(context.Background(), stub, "Lisa", "故事", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Decision != model.TopicDecisionNewTopicEvent || got.Status != model.EventStatusPending {
		t.Fatalf("decision = %+v", got)
	}
}

func TestMatchTopicEventRejectsUnavailableOrIncompatibleReuse(t *testing.T) {
	topics, events := matcherCandidates(model.EventStatusActive)
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "invented ID",
			output: `{"decision":"reuse_event","relation":"continuation","topic_id":"T-999999","event_id":"E-999999","topic_summary":"","event_summary":"","event_type":"","subject":"","target":"","status":"","reason":"相同事件。"}`,
			want:   "unavailable",
		},
		{
			name:   "confirmation of active event",
			output: `{"decision":"reuse_event","relation":"confirmation","topic_id":"T-000001","event_id":"E-000001","topic_summary":"","event_summary":"","event_type":"","subject":"","target":"","status":"","reason":"事件確認。"}`,
			want:   "requires a pending event",
		},
		{
			name:   "reuse leaks new fields",
			output: `{"decision":"reuse_event","relation":"continuation","topic_id":"T-000001","event_id":"E-000001","topic_summary":"不該存在","event_summary":"","event_type":"","subject":"","target":"","status":"","reason":"相同事件。"}`,
			want:   "reserved",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &topicMatcherStub{result: tt.output}
			_, err := MatchTopicEvent(context.Background(), stub, "Lisa", "故事", topics, events)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestMatchTopicEventRejectsMoreThan32Candidates(t *testing.T) {
	topics := []model.Topic{{ID: "T-000001"}}
	events := make([]model.NarrativeEvent, TopicMatcherMaxCandidates+1)
	for i := range events {
		events[i] = model.NarrativeEvent{
			ID:      fmt.Sprintf("E-%06d", i+1),
			TopicID: "T-000001",
			Status:  model.EventStatusActive,
		}
	}
	_, err := MatchTopicEvent(context.Background(), &topicMatcherStub{}, "Lisa", "故事", topics, events)
	if err == nil || !strings.Contains(err.Error(), "maximum is 32") {
		t.Fatalf("error = %v", err)
	}
}

func TestMatchTopicEventSupportsNoopClient(t *testing.T) {
	got, err := MatchTopicEvent(context.Background(), NewNoopClient(), "Lisa", "故事", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Decision != model.TopicDecisionNewTopicEvent || got.Subject != "Lisa" {
		t.Fatalf("decision = %+v", got)
	}
}

func TestMatchTopicEventWrapsProviderError(t *testing.T) {
	want := errors.New("provider unavailable")
	_, err := MatchTopicEvent(context.Background(), &topicMatcherStub{err: want}, "Lisa", "故事", nil, nil)
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want wrapping %v", err, want)
	}
}

func matcherCandidates(status model.EventStatus) ([]model.Topic, []model.NarrativeEvent) {
	now := time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
	return []model.Topic{{
			ID:               "T-000001",
			CharacterID:      "lisa",
			CanonicalSummary: "玩家與 Lisa 的承諾",
			Participants:     []string{"玩家", "Lisa"},
			CreatedAt:        now,
			LastSeenAt:       now,
		}}, []model.NarrativeEvent{{
			ID:               "E-000001",
			TopicID:          "T-000001",
			CharacterID:      "lisa",
			CanonicalSummary: "玩家承諾返回 Lisa 身邊",
			EventType:        "promise",
			Subject:          "玩家",
			Target:           "Lisa",
			Status:           status,
			CreatedAt:        now,
			LastSeenAt:       now,
		}}
}
