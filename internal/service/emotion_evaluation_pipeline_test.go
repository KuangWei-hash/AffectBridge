package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
)

type evaluationLLMStub struct {
	mu           sync.Mutex
	calls        int
	blockWorkers bool
}

func (s *evaluationLLMStub) Complete(ctx context.Context, prompt string, _ ...llm.Option) (string, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(prompt), &object); err != nil {
		return "", err
	}
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	if _, ok := object["dialogue"]; ok {
		return "玩家送花給Lisa。", nil
	}
	if _, ok := object["events"]; ok {
		return `{
          "decision":"new_topic_event","relation":"new",
          "topic_id":"","event_id":"",
          "topic_summary":"玩家與Lisa的送禮互動",
          "event_summary":"玩家送花給Lisa","event_type":"gift",
          "subject":"玩家","target":"Lisa","status":"closed",
          "reason":"候選中沒有這次送禮事件。"
        }`, nil
	}
	if rawTag, ok := object["target_tag"]; ok {
		if s.blockWorkers {
			<-ctx.Done()
			return "", ctx.Err()
		}
		var tag model.AppraisalTag
		if err := json.Unmarshal(rawTag, &tag); err != nil {
			return "", err
		}
		matched := tag == model.GoodEvent
		intensity := model.IntensityNotApplicable
		reason := "沒有足夠證據。"
		if matched {
			intensity = model.IntensityStrong
			reason = "玩家送花對Lisa有利。"
		}
		return fmt.Sprintf(`{"tag":%q,"matched":%t,"reason":%q,"intensity":%q}`,
			tag, matched, reason, intensity), nil
	}
	return "", fmt.Errorf("unknown prompt: %s", prompt)
}

func TestEmotionEvaluationPipelineRunsThroughALMADispatch(t *testing.T) {
	llmClient := &evaluationLLMStub{}
	sender := &appraisalSenderStub{}
	pipeline, topicRepo := newEvaluationPipeline(llmClient, sender)
	result, err := pipeline.Evaluate(context.Background(), "lisa", "Lisa", "Lisa 重視真誠的禮物。", "送花給Lisa")
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Story.Text != "玩家送花給Lisa。" || len(result.Appraisals.Results) != model.AppraisalTagCount {
		t.Fatalf("result = %+v", result)
	}
	if result.Topic.Event.ID != "E-000001" || len(result.Dispatch.Receipts) != 1 {
		t.Fatalf("topic=%+v dispatch=%+v", result.Topic, result.Dispatch)
	}
	if len(sender.signals) != 1 || sender.signals[0].Tag != model.GoodEvent ||
		sender.signals[0].Intensity != 0.65 || sender.signals[0].Elicitor != "E-000001" {
		t.Fatalf("signals = %+v", sender.signals)
	}
	llmClient.mu.Lock()
	calls := llmClient.calls
	llmClient.mu.Unlock()
	if calls != 20 {
		t.Fatalf("LLM calls = %d, want 20 (Story + Topic + 18 workers)", calls)
	}
	snapshot, err := topicRepo.Candidates("lisa", repository.TopicEventPoolCapacity)
	if err != nil || len(snapshot.Events) != 1 {
		t.Fatalf("pool snapshot = %+v, error = %v", snapshot, err)
	}
}

func TestEmotionEvaluationPipelineTimeoutDoesNotCommitOrDispatch(t *testing.T) {
	llmClient := &evaluationLLMStub{blockWorkers: true}
	sender := &appraisalSenderStub{}
	pipeline, topicRepo := newEvaluationPipeline(llmClient, sender)
	pipeline.timeout = 30 * time.Millisecond
	_, err := pipeline.Evaluate(context.Background(), "lisa", "Lisa", "", "送花給Lisa")
	if err == nil {
		t.Fatal("Evaluate() succeeded, want timeout")
	}
	snapshot, snapshotErr := topicRepo.Candidates("lisa", repository.TopicEventPoolCapacity)
	if snapshotErr != nil {
		t.Fatal(snapshotErr)
	}
	if len(snapshot.Events) != 0 || len(sender.signals) != 0 {
		t.Fatalf("timeout mutated pool or ALMA: events=%d signals=%d", len(snapshot.Events), len(sender.signals))
	}
}

func TestEmotionEvaluationPipelineDefaultTimeoutIsOneMinute(t *testing.T) {
	pipeline, _ := newEvaluationPipeline(&evaluationLLMStub{}, &appraisalSenderStub{})
	if pipeline.timeout != time.Minute || EmotionEvaluationTimeout != time.Minute {
		t.Fatalf("timeout=%v constant=%v, want one minute", pipeline.timeout, EmotionEvaluationTimeout)
	}
}

func newEvaluationPipeline(
	llmClient llm.Client,
	sender *appraisalSenderStub,
) (*EmotionEvaluationPipeline, *repository.InMemoryTopicEventRepository) {
	dialogueRepo := repository.NewInMemoryDialogueRepository(64)
	storyPipeline := NewStoryPipeline(NewStoryService(llmClient), dialogueRepo)
	topicRepo := repository.NewInMemoryTopicEventRepository()
	topicService := NewTopicMatcherService(llmClient, topicRepo)
	workerService := NewAppraisalWorkerService(llmClient)
	dispatcher := NewBasicAppraisalDispatcher(sender)
	return NewEmotionEvaluationPipeline(storyPipeline, topicService, workerService, dispatcher), topicRepo
}
