package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

const AppraisalWorkerBatchTimeout = time.Minute

// AppraisalWorkerService fans one immutable Story out to all 18 specialized
// workers. It intentionally contains no semaphore, worker pool, or queue.
type AppraisalWorkerService struct {
	llm     llm.Client
	timeout time.Duration
}

func NewAppraisalWorkerService(llmClient llm.Client) *AppraisalWorkerService {
	return &AppraisalWorkerService{
		llm:     llmClient,
		timeout: AppraisalWorkerBatchTimeout,
	}
}

type AppraisalWorkerFailure struct {
	Tag model.AppraisalTag
	Err error
}

// AppraisalWorkerBatchError reports every failed worker in canonical tag order.
// Successful results remain available in the returned AppraisalAnalysis.
type AppraisalWorkerBatchError struct {
	Failures []AppraisalWorkerFailure
}

func (e *AppraisalWorkerBatchError) Error() string {
	parts := make([]string, 0, len(e.Failures))
	for _, failure := range e.Failures {
		parts = append(parts, fmt.Sprintf("%s: %v", failure.Tag, failure.Err))
	}
	return fmt.Sprintf("appraisal workers: %d of %d failed: %s",
		len(e.Failures), model.AppraisalTagCount, strings.Join(parts, "; "))
}

func (e *AppraisalWorkerBatchError) Unwrap() []error {
	errs := make([]error, 0, len(e.Failures))
	for _, failure := range e.Failures {
		errs = append(errs, failure.Err)
	}
	return errs
}

type appraisalWorkerOutcome struct {
	result model.AppraisalWorkerResult
	err    error
}

// Analyze releases all 18 worker goroutines through one start gate. Therefore
// every request is eligible to enter Client.Complete at the same time; this
// service never intentionally queues one appraisal tag behind another.
//
// The supplied Client must be safe for concurrent use and must not be wrapped
// in a Limiter smaller than 18. Analyze waits for every worker so it can return
// all successful results and all failures together. One shared one-minute
// timeout bounds the entire batch; when it expires, all in-flight calls are
// cancelled and the entire analysis result is discarded.
func (s *AppraisalWorkerService) Analyze(
	ctx context.Context,
	characterName string,
	characterView string,
	story model.Story,
) (model.AppraisalAnalysis, error) {
	if s == nil || s.llm == nil {
		return model.AppraisalAnalysis{}, fmt.Errorf("appraisal workers: llm client is nil")
	}
	if strings.TrimSpace(characterName) == "" {
		return model.AppraisalAnalysis{}, llm.ErrEmptyCharacterName
	}
	if strings.TrimSpace(story.Text) == "" {
		return model.AppraisalAnalysis{}, llm.ErrEmptyAppraisalStory
	}
	timeout := s.timeout
	if timeout <= 0 {
		timeout = AppraisalWorkerBatchTimeout
	}
	workCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var outcomes [model.AppraisalTagCount]appraisalWorkerOutcome
	start := make(chan struct{})
	var ready sync.WaitGroup
	var finished sync.WaitGroup
	ready.Add(model.AppraisalTagCount)
	finished.Add(model.AppraisalTagCount)

	for i, tag := range model.AllAppraisalTags {
		go func(index int, workerTag model.AppraisalTag) {
			defer finished.Done()
			ready.Done()
			select {
			case <-start:
			case <-workCtx.Done():
				outcomes[index].err = workCtx.Err()
				return
			}
			outcomes[index].result, outcomes[index].err = llm.RunAppraisalWorker(
				workCtx,
				s.llm,
				workerTag,
				characterName,
				characterView,
				story.Text,
			)
		}(i, tag)
	}

	ready.Wait()
	close(start)
	allFinished := make(chan struct{})
	go func() {
		finished.Wait()
		close(allFinished)
	}()
	select {
	case <-allFinished:
		if err := workCtx.Err(); err != nil {
			return model.AppraisalAnalysis{}, fmt.Errorf("appraisal workers: batch abandoned: %w", err)
		}
	case <-workCtx.Done():
		return model.AppraisalAnalysis{}, fmt.Errorf("appraisal workers: batch abandoned: %w", workCtx.Err())
	}

	analysis := model.AppraisalAnalysis{
		Results: make([]model.AppraisalWorkerResult, 0, model.AppraisalTagCount),
	}
	batchErr := &AppraisalWorkerBatchError{
		Failures: make([]AppraisalWorkerFailure, 0),
	}
	for i, tag := range model.AllAppraisalTags {
		outcome := outcomes[i]
		if outcome.err != nil {
			batchErr.Failures = append(batchErr.Failures, AppraisalWorkerFailure{
				Tag: tag,
				Err: outcome.err,
			})
			continue
		}
		analysis.Results = append(analysis.Results, outcome.result)
	}
	if len(batchErr.Failures) > 0 {
		return analysis, batchErr
	}
	return analysis, nil
}
