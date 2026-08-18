package repository

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

const TopicEventPoolCapacity = 32

var (
	ErrTopicNotFound     = errors.New("topic event repository: topic not found")
	ErrEventNotFound     = errors.New("topic event repository: event not found")
	ErrEventPoolFull     = errors.New("topic event repository: all 32 event slots are protected")
	ErrTopicPoolConflict = errors.New("topic event repository: pool changed after proposal")
)

type TopicEventSnapshot struct {
	Version uint64                 `json:"version"`
	Topics  []model.Topic          `json:"topics"`
	Events  []model.NarrativeEvent `json:"events"`
}

// TopicEventRepository owns stable Topic and Event identities. Apply performs
// one validated decision atomically from the repository's point of view.
type TopicEventRepository interface {
	Candidates(characterID string, limit int) (TopicEventSnapshot, error)
	Apply(characterID string, decision model.TopicMatchDecision) (model.TopicResolution, error)
	ApplyIfVersion(characterID string, expectedVersion uint64, decision model.TopicMatchDecision) (model.TopicResolution, error)
}

type InMemoryTopicEventRepository struct {
	mu        sync.RWMutex
	topics    map[string]map[string]model.Topic
	events    map[string]map[string]model.NarrativeEvent
	versions  map[string]uint64
	nextTopic uint64
	nextEvent uint64
	now       func() time.Time
}

func NewInMemoryTopicEventRepository() *InMemoryTopicEventRepository {
	return &InMemoryTopicEventRepository{
		topics:   make(map[string]map[string]model.Topic),
		events:   make(map[string]map[string]model.NarrativeEvent),
		versions: make(map[string]uint64),
		now:      time.Now,
	}
}

func (r *InMemoryTopicEventRepository) Candidates(characterID string, limit int) (TopicEventSnapshot, error) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return TopicEventSnapshot{}, errors.New("topic event repository: character ID is empty")
	}
	if limit < 1 || limit > TopicEventPoolCapacity {
		return TopicEventSnapshot{}, fmt.Errorf("topic event repository: candidate limit must be between 1 and %d", TopicEventPoolCapacity)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	events := make([]model.NarrativeEvent, 0, len(r.events[characterID]))
	for _, event := range r.events[characterID] {
		events = append(events, cloneNarrativeEvent(event))
	}
	sort.Slice(events, func(i, j int) bool {
		leftPending := events[i].Status == model.EventStatusPending
		rightPending := events[j].Status == model.EventStatusPending
		if leftPending != rightPending {
			return leftPending
		}
		if !events[i].LastSeenAt.Equal(events[j].LastSeenAt) {
			return events[i].LastSeenAt.After(events[j].LastSeenAt)
		}
		return events[i].ID < events[j].ID
	})
	if len(events) > limit {
		events = events[:limit]
	}

	topicIDs := make(map[string]struct{}, len(events))
	for _, event := range events {
		topicIDs[event.TopicID] = struct{}{}
	}
	topics := make([]model.Topic, 0, len(topicIDs))
	for topicID := range topicIDs {
		if topic, ok := r.topics[characterID][topicID]; ok {
			topics = append(topics, cloneTopic(topic))
		}
	}
	sort.Slice(topics, func(i, j int) bool { return topics[i].ID < topics[j].ID })
	return TopicEventSnapshot{Version: r.versions[characterID], Topics: topics, Events: events}, nil
}

func (r *InMemoryTopicEventRepository) Apply(
	characterID string,
	decision model.TopicMatchDecision,
) (model.TopicResolution, error) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return model.TopicResolution{}, errors.New("topic event repository: character ID is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.applyLocked(characterID, decision)
}

func (r *InMemoryTopicEventRepository) ApplyIfVersion(
	characterID string,
	expectedVersion uint64,
	decision model.TopicMatchDecision,
) (model.TopicResolution, error) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return model.TopicResolution{}, errors.New("topic event repository: character ID is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.versions[characterID] != expectedVersion {
		return model.TopicResolution{}, ErrTopicPoolConflict
	}
	return r.applyLocked(characterID, decision)
}

func (r *InMemoryTopicEventRepository) applyLocked(
	characterID string,
	decision model.TopicMatchDecision,
) (model.TopicResolution, error) {
	if r.topics[characterID] == nil {
		r.topics[characterID] = make(map[string]model.Topic)
	}
	if r.events[characterID] == nil {
		r.events[characterID] = make(map[string]model.NarrativeEvent)
	}
	now := r.now().UTC()

	var resolution model.TopicResolution
	var err error
	switch decision.Decision {
	case model.TopicDecisionReuseEvent:
		resolution, err = r.reuseEventLocked(characterID, decision, now)
	case model.TopicDecisionNewEvent:
		resolution, err = r.newEventLocked(characterID, decision, now)
	case model.TopicDecisionNewTopicEvent:
		resolution, err = r.newTopicEventLocked(characterID, decision, now)
	default:
		return model.TopicResolution{}, fmt.Errorf("topic event repository: invalid decision %q", decision.Decision)
	}
	if err != nil {
		return model.TopicResolution{}, err
	}
	r.versions[characterID]++
	return resolution, nil
}

func (r *InMemoryTopicEventRepository) reuseEventLocked(
	characterID string,
	decision model.TopicMatchDecision,
	now time.Time,
) (model.TopicResolution, error) {
	topic, ok := r.topics[characterID][decision.TopicID]
	if !ok {
		return model.TopicResolution{}, ErrTopicNotFound
	}
	event, ok := r.events[characterID][decision.EventID]
	if !ok || event.TopicID != topic.ID {
		return model.TopicResolution{}, ErrEventNotFound
	}
	if decision.Relation != model.EventRelationContinuation &&
		decision.Relation != model.EventRelationConfirmation &&
		decision.Relation != model.EventRelationDisconfirmation {
		return model.TopicResolution{}, fmt.Errorf("topic event repository: invalid reuse relation %q", decision.Relation)
	}
	if decision.Relation == model.EventRelationConfirmation || decision.Relation == model.EventRelationDisconfirmation {
		if event.Status != model.EventStatusPending {
			return model.TopicResolution{}, fmt.Errorf("topic event repository: %s requires pending event, got %s", decision.Relation, event.Status)
		}
		resolvedAt := now
		event.ResolvedAt = &resolvedAt
		if decision.Relation == model.EventRelationConfirmation {
			event.Status = model.EventStatusRealized
		} else {
			event.Status = model.EventStatusDisconfirmed
		}
	}
	event.LastSeenAt = now
	topic.LastSeenAt = now
	r.events[characterID][event.ID] = event
	r.topics[characterID][topic.ID] = topic
	return model.TopicResolution{
		Topic:    cloneTopic(topic),
		Event:    cloneNarrativeEvent(event),
		Created:  false,
		Relation: decision.Relation,
		Reason:   strings.TrimSpace(decision.Reason),
	}, nil
}

func (r *InMemoryTopicEventRepository) newEventLocked(
	characterID string,
	decision model.TopicMatchDecision,
	now time.Time,
) (model.TopicResolution, error) {
	topic, ok := r.topics[characterID][decision.TopicID]
	if !ok {
		return model.TopicResolution{}, ErrTopicNotFound
	}
	if err := validateNewEventDecision(decision); err != nil {
		return model.TopicResolution{}, err
	}
	if err := r.makeEventSlotLocked(characterID, topic.ID); err != nil {
		return model.TopicResolution{}, err
	}
	event := r.createEventLocked(characterID, topic.ID, decision, now)
	topic.Participants = mergeParticipants(topic.Participants, event.Subject, event.Target)
	topic.LastSeenAt = now
	r.topics[characterID][topic.ID] = topic
	return model.TopicResolution{
		Topic:    cloneTopic(topic),
		Event:    cloneNarrativeEvent(event),
		Created:  true,
		Relation: model.EventRelationNew,
		Reason:   strings.TrimSpace(decision.Reason),
	}, nil
}

func (r *InMemoryTopicEventRepository) newTopicEventLocked(
	characterID string,
	decision model.TopicMatchDecision,
	now time.Time,
) (model.TopicResolution, error) {
	if err := validateNewEventDecision(decision); err != nil {
		return model.TopicResolution{}, err
	}
	if strings.TrimSpace(decision.TopicSummary) == "" {
		return model.TopicResolution{}, errors.New("topic event repository: new topic summary is empty")
	}
	if err := r.makeEventSlotLocked(characterID, ""); err != nil {
		return model.TopicResolution{}, err
	}
	r.nextTopic++
	topic := model.Topic{
		ID:               fmt.Sprintf("T-%06d", r.nextTopic),
		CharacterID:      characterID,
		CanonicalSummary: strings.TrimSpace(decision.TopicSummary),
		Participants:     mergeParticipants(nil, decision.Subject, decision.Target),
		CreatedAt:        now,
		LastSeenAt:       now,
	}
	r.topics[characterID][topic.ID] = topic
	event := r.createEventLocked(characterID, topic.ID, decision, now)
	return model.TopicResolution{
		Topic:    cloneTopic(topic),
		Event:    cloneNarrativeEvent(event),
		Created:  true,
		Relation: model.EventRelationNew,
		Reason:   strings.TrimSpace(decision.Reason),
	}, nil
}

func validateNewEventDecision(decision model.TopicMatchDecision) error {
	if decision.Relation != model.EventRelationNew {
		return fmt.Errorf("topic event repository: new event has relation %q", decision.Relation)
	}
	if strings.TrimSpace(decision.EventSummary) == "" ||
		strings.TrimSpace(decision.EventType) == "" ||
		strings.TrimSpace(decision.Subject) == "" ||
		strings.TrimSpace(decision.Target) == "" {
		return errors.New("topic event repository: new event fields are incomplete")
	}
	if decision.Status != model.EventStatusPending &&
		decision.Status != model.EventStatusActive &&
		decision.Status != model.EventStatusClosed {
		return fmt.Errorf("topic event repository: invalid new event status %q", decision.Status)
	}
	return nil
}

func (r *InMemoryTopicEventRepository) createEventLocked(
	characterID string,
	topicID string,
	decision model.TopicMatchDecision,
	now time.Time,
) model.NarrativeEvent {
	r.nextEvent++
	event := model.NarrativeEvent{
		ID:               fmt.Sprintf("E-%06d", r.nextEvent),
		TopicID:          topicID,
		CharacterID:      characterID,
		CanonicalSummary: strings.TrimSpace(decision.EventSummary),
		EventType:        strings.TrimSpace(decision.EventType),
		Subject:          strings.TrimSpace(decision.Subject),
		Target:           strings.TrimSpace(decision.Target),
		Status:           decision.Status,
		CreatedAt:        now,
		LastSeenAt:       now,
	}
	r.events[characterID][event.ID] = event
	return event
}

func (r *InMemoryTopicEventRepository) makeEventSlotLocked(characterID, keepTopicID string) error {
	if len(r.events[characterID]) < TopicEventPoolCapacity {
		return nil
	}
	var selected model.NarrativeEvent
	selectedRank := 0
	found := false
	for _, event := range r.events[characterID] {
		rank, evictable := eventEvictionRank(event.Status)
		if !evictable {
			continue
		}
		if !found || rank < selectedRank ||
			(rank == selectedRank && event.LastSeenAt.Before(selected.LastSeenAt)) ||
			(rank == selectedRank && event.LastSeenAt.Equal(selected.LastSeenAt) && event.ID < selected.ID) {
			selected = event
			selectedRank = rank
			found = true
		}
	}
	if !found {
		return ErrEventPoolFull
	}
	delete(r.events[characterID], selected.ID)
	if selected.TopicID != keepTopicID && !r.topicHasEventsLocked(characterID, selected.TopicID) {
		delete(r.topics[characterID], selected.TopicID)
	}
	return nil
}

func eventEvictionRank(status model.EventStatus) (int, bool) {
	switch status {
	case model.EventStatusClosed:
		return 0, true
	case model.EventStatusRealized, model.EventStatusDisconfirmed:
		return 1, true
	case model.EventStatusActive:
		return 2, true
	default:
		return 0, false
	}
}

func (r *InMemoryTopicEventRepository) topicHasEventsLocked(characterID, topicID string) bool {
	for _, event := range r.events[characterID] {
		if event.TopicID == topicID {
			return true
		}
	}
	return false
}

func mergeParticipants(existing []string, names ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(names))
	result := make([]string, 0, len(existing)+len(names))
	for _, name := range append(append([]string(nil), existing...), names...) {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func cloneTopic(topic model.Topic) model.Topic {
	topic.Participants = append([]string(nil), topic.Participants...)
	return topic
}

func cloneNarrativeEvent(event model.NarrativeEvent) model.NarrativeEvent {
	if event.ResolvedAt != nil {
		resolvedAt := *event.ResolvedAt
		event.ResolvedAt = &resolvedAt
	}
	return event
}
