package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

type simultaneousWorkerClient struct {
	mu         sync.Mutex
	started    int
	active     int
	maxActive  int
	allStarted chan struct{}
	release    chan struct{}
	failTag    model.AppraisalTag
	failErr    error
}

func newSimultaneousWorkerClient() *simultaneousWorkerClient {
	return &simultaneousWorkerClient{
		allStarted: make(chan struct{}),
		release:    make(chan struct{}),
	}
}

func (c *simultaneousWorkerClient) Complete(ctx context.Context, prompt string, _ ...llm.Option) (string, error) {
	var input struct {
		TargetTag model.AppraisalTag `json:"target_tag"`
	}
	if err := json.Unmarshal([]byte(prompt), &input); err != nil {
		return "", err
	}

	c.mu.Lock()
	c.started++
	c.active++
	if c.active > c.maxActive {
		c.maxActive = c.active
	}
	if c.started == model.AppraisalTagCount {
		close(c.allStarted)
	}
	c.mu.Unlock()

	select {
	case <-c.release:
	case <-ctx.Done():
		c.finish()
		return "", ctx.Err()
	}
	c.finish()
	if input.TargetTag == c.failTag {
		return "", c.failErr
	}
	return fmt.Sprintf(`{"tag":%q,"matched":false,"reason":"沒有足夠證據。","intensity":"不適用"}`, input.TargetTag), nil
}

func (c *simultaneousWorkerClient) finish() {
	c.mu.Lock()
	c.active--
	c.mu.Unlock()
}

func TestAppraisalWorkerServiceStartsAll18Together(t *testing.T) {
	client := newSimultaneousWorkerClient()
	service := NewAppraisalWorkerService(client)
	type response struct {
		analysis model.AppraisalAnalysis
		err      error
	}
	done := make(chan response, 1)
	go func() {
		analysis, err := service.Analyze(context.Background(), "Lisa", "Lisa 重視承諾。", model.Story{Text: "玩家履行承諾。"})
		done <- response{analysis: analysis, err: err}
	}()

	select {
	case <-client.allStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("not all 18 workers entered Complete before any response was released")
	}
	client.mu.Lock()
	started, maxActive := client.started, client.maxActive
	client.mu.Unlock()
	if started != model.AppraisalTagCount || maxActive != model.AppraisalTagCount {
		t.Fatalf("started=%d maxActive=%d, want 18 and 18", started, maxActive)
	}
	close(client.release)

	got := <-done
	if got.err != nil {
		t.Fatalf("Analyze() error = %v", got.err)
	}
	if len(got.analysis.Results) != model.AppraisalTagCount {
		t.Fatalf("results = %d, want 18", len(got.analysis.Results))
	}
	for i, result := range got.analysis.Results {
		if result.Tag != model.AllAppraisalTags[i] {
			t.Fatalf("result[%d].Tag = %s, want %s", i, result.Tag, model.AllAppraisalTags[i])
		}
	}
}

func TestAppraisalWorkerServiceReturnsPartialResultsAndAllFailures(t *testing.T) {
	want := errors.New("worker failed")
	client := newSimultaneousWorkerClient()
	client.failTag = model.BadEvent
	client.failErr = want
	service := NewAppraisalWorkerService(client)
	done := make(chan struct{})
	var analysis model.AppraisalAnalysis
	var err error
	go func() {
		analysis, err = service.Analyze(context.Background(), "Lisa", "", model.Story{Text: "故事"})
		close(done)
	}()
	<-client.allStarted
	close(client.release)
	<-done

	if len(analysis.Results) != model.AppraisalTagCount-1 {
		t.Fatalf("partial results = %d, want 17", len(analysis.Results))
	}
	var batchErr *AppraisalWorkerBatchError
	if !errors.As(err, &batchErr) {
		t.Fatalf("error = %v, want AppraisalWorkerBatchError", err)
	}
	if len(batchErr.Failures) != 1 || batchErr.Failures[0].Tag != model.BadEvent {
		t.Fatalf("failures = %+v", batchErr.Failures)
	}
	if !errors.Is(err, want) {
		t.Fatalf("error does not wrap provider failure: %v", err)
	}
}

func TestAppraisalWorkerServiceAbandonsWholeBatchOnTimeout(t *testing.T) {
	client := newSimultaneousWorkerClient()
	service := NewAppraisalWorkerService(client)
	service.timeout = 25 * time.Millisecond

	startedAt := time.Now()
	analysis, err := service.Analyze(context.Background(), "Lisa", "", model.Story{Text: "故事"})
	elapsed := time.Since(startedAt)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
	if len(analysis.Results) != 0 {
		t.Fatalf("timeout returned %d partial results, want none", len(analysis.Results))
	}
	if elapsed > time.Second {
		t.Fatalf("timeout returned after %v, want prompt abandonment", elapsed)
	}
	select {
	case <-client.allStarted:
	default:
		t.Fatal("timeout occurred before all 18 workers were released together")
	}
}

func TestAppraisalWorkerServiceDefaultTimeoutIsOneMinute(t *testing.T) {
	service := NewAppraisalWorkerService(newSimultaneousWorkerClient())
	if service.timeout != time.Minute || AppraisalWorkerBatchTimeout != time.Minute {
		t.Fatalf("timeout = %v, constant = %v; want one minute", service.timeout, AppraisalWorkerBatchTimeout)
	}
}

func TestAppraisalWorkerServiceRejectsInvalidInputBeforeFanout(t *testing.T) {
	client := newSimultaneousWorkerClient()
	service := NewAppraisalWorkerService(client)
	_, err := service.Analyze(context.Background(), "", "", model.Story{Text: "故事"})
	if !errors.Is(err, llm.ErrEmptyCharacterName) {
		t.Fatalf("error = %v, want ErrEmptyCharacterName", err)
	}
	_, err = service.Analyze(context.Background(), "Lisa", "", model.Story{})
	if !errors.Is(err, llm.ErrEmptyAppraisalStory) {
		t.Fatalf("error = %v, want ErrEmptyAppraisalStory", err)
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.started != 0 {
		t.Fatalf("started = %d, want 0", client.started)
	}
}
