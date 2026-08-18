package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
)

// TopicMatcherService resolves one primary Event identity before appraisal
// fan-out. Resolutions for the same character are serialized so two concurrent
// Stories cannot both read the same snapshot and create duplicate Events.
type TopicMatcherService struct {
	llm   llm.Client
	repo  repository.TopicEventRepository
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewTopicMatcherService(
	llmClient llm.Client,
	repo repository.TopicEventRepository,
) *TopicMatcherService {
	return &TopicMatcherService{
		llm:   llmClient,
		repo:  repo,
		locks: make(map[string]*sync.Mutex),
	}
}

// Propose asks the LLM to choose an offered Event or request a new one without
// mutating the Pool. The caller commits only after every other pipeline stage
// succeeds inside the shared top-level deadline.
func (s *TopicMatcherService) Propose(
	ctx context.Context,
	characterID string,
	characterName string,
	story model.Story,
) (model.TopicMatchProposal, error) {
	if s == nil || s.llm == nil || s.repo == nil {
		return model.TopicMatchProposal{}, errors.New("topic matcher service: dependencies are nil")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return model.TopicMatchProposal{}, errors.New("topic matcher service: character ID is empty")
	}
	if err := ctx.Err(); err != nil {
		return model.TopicMatchProposal{}, err
	}
	snapshot, err := s.repo.Candidates(characterID, llm.TopicMatcherMaxCandidates)
	if err != nil {
		return model.TopicMatchProposal{}, err
	}
	decision, err := llm.MatchTopicEvent(
		ctx,
		s.llm,
		characterName,
		story.Text,
		snapshot.Topics,
		snapshot.Events,
	)
	if err != nil {
		return model.TopicMatchProposal{}, err
	}
	if err := ctx.Err(); err != nil {
		return model.TopicMatchProposal{}, err
	}
	return model.TopicMatchProposal{
		CharacterID: characterID,
		PoolVersion: snapshot.Version,
		Decision:    decision,
	}, nil
}

// Commit atomically applies a proposal if no other resolution has changed the
// character's Pool since the proposal was created.
func (s *TopicMatcherService) Commit(proposal model.TopicMatchProposal) (model.TopicResolution, error) {
	if s == nil || s.repo == nil {
		return model.TopicResolution{}, errors.New("topic matcher service: repository is nil")
	}
	return s.repo.ApplyIfVersion(proposal.CharacterID, proposal.PoolVersion, proposal.Decision)
}

// Resolve is a convenience operation for callers whose whole task consists of
// matching alone. The structured appraisal pipeline must use Propose and defer
// Commit until all 18 workers have completed successfully.
func (s *TopicMatcherService) Resolve(
	ctx context.Context,
	characterID string,
	characterName string,
	story model.Story,
) (model.TopicResolution, error) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return model.TopicResolution{}, errors.New("topic matcher service: character ID is empty")
	}
	lock := s.characterLock(characterID)
	lock.Lock()
	defer lock.Unlock()
	proposal, err := s.Propose(ctx, characterID, characterName, story)
	if err != nil {
		return model.TopicResolution{}, err
	}
	if err := ctx.Err(); err != nil {
		return model.TopicResolution{}, err
	}
	return s.Commit(proposal)
}

func (s *TopicMatcherService) characterLock(characterID string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock := s.locks[characterID]
	if lock == nil {
		lock = &sync.Mutex{}
		s.locks[characterID] = lock
	}
	return lock
}
