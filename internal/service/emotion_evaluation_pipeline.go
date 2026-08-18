package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

const EmotionEvaluationTimeout = time.Minute

// EmotionEvaluationPipeline is the top-level structured appraisal path:
// dialogue -> Story -> Topic proposal -> 18 workers -> Event commit -> ALMA.
// Evaluations for the same character are serialized, while different
// characters remain independent.
type EmotionEvaluationPipeline struct {
	stories    *StoryPipeline
	topics     *TopicMatcherService
	workers    *AppraisalWorkerService
	dispatcher *BasicAppraisalDispatcher
	mu         sync.Mutex
	locks      map[string]chan struct{}
	timeout    time.Duration
}

func NewEmotionEvaluationPipeline(
	stories *StoryPipeline,
	topics *TopicMatcherService,
	workers *AppraisalWorkerService,
	dispatcher *BasicAppraisalDispatcher,
) *EmotionEvaluationPipeline {
	return &EmotionEvaluationPipeline{
		stories:    stories,
		topics:     topics,
		workers:    workers,
		dispatcher: dispatcher,
		locks:      make(map[string]chan struct{}),
		timeout:    EmotionEvaluationTimeout,
	}
}

type EmotionEvaluationResult struct {
	Story      model.Story             `json:"story"`
	Topic      model.TopicResolution   `json:"topic"`
	Appraisals model.AppraisalAnalysis `json:"appraisals"`
	Dispatch   AppraisalDispatchReport `json:"dispatch"`
}

// Evaluate runs the complete affect-input transaction under one shared
// one-minute deadline. Dialogue history is not committed here; the chat caller
// commits the exchange only after reply generation succeeds.
func (p *EmotionEvaluationPipeline) Evaluate(
	ctx context.Context,
	characterID string,
	characterName string,
	characterView string,
	playerMessage string,
) (EmotionEvaluationResult, error) {
	if p == nil || p.stories == nil || p.topics == nil || p.workers == nil || p.dispatcher == nil {
		return EmotionEvaluationResult{}, errors.New("emotion evaluation: dependencies are nil")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return EmotionEvaluationResult{}, errors.New("emotion evaluation: character ID is empty")
	}
	timeout := p.timeout
	if timeout <= 0 {
		timeout = EmotionEvaluationTimeout
	}
	workCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	release, err := p.acquireCharacter(workCtx, characterID)
	if err != nil {
		return EmotionEvaluationResult{}, err
	}
	defer release()

	var result EmotionEvaluationResult
	result.Story, err = p.stories.BuildForPlayerMessage(
		workCtx,
		characterID,
		characterName,
		playerMessage,
	)
	if err != nil {
		return result, err
	}
	proposal, err := p.topics.Propose(
		workCtx,
		characterID,
		characterName,
		result.Story,
	)
	if err != nil {
		return result, err
	}
	result.Appraisals, err = p.workers.Analyze(
		workCtx,
		characterName,
		characterView,
		result.Story,
	)
	if err != nil {
		return result, err
	}
	if err := workCtx.Err(); err != nil {
		return result, err
	}
	result.Topic, err = p.topics.Commit(proposal)
	if err != nil {
		return result, err
	}
	if err := workCtx.Err(); err != nil {
		return result, err
	}
	result.Dispatch, err = p.dispatcher.Dispatch(
		workCtx,
		characterName,
		result.Topic.Event.ID,
		result.Appraisals,
	)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (p *EmotionEvaluationPipeline) acquireCharacter(
	ctx context.Context,
	characterID string,
) (func(), error) {
	p.mu.Lock()
	lock := p.locks[characterID]
	if lock == nil {
		lock = make(chan struct{}, 1)
		p.locks[characterID] = lock
	}
	p.mu.Unlock()
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
