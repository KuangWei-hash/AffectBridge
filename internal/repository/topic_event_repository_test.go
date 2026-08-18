package repository

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

func TestTopicEventRepositoryCreatesAndConfirmsPendingEvent(t *testing.T) {
	repo := NewInMemoryTopicEventRepository()
	now := time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
	repo.now = func() time.Time {
		now = now.Add(time.Second)
		return now
	}
	created, err := repo.Apply("lisa", newTopicDecision("玩家承諾返回", model.EventStatusPending))
	if err != nil {
		t.Fatal(err)
	}
	if created.Topic.ID != "T-000001" || created.Event.ID != "E-000001" || !created.Created {
		t.Fatalf("created = %+v", created)
	}

	confirmed, err := repo.Apply("lisa", model.TopicMatchDecision{
		Decision: model.TopicDecisionReuseEvent,
		Relation: model.EventRelationConfirmation,
		TopicID:  created.Topic.ID,
		EventID:  created.Event.ID,
		Reason:   "玩家真的返回。",
	})
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Created || confirmed.Event.Status != model.EventStatusRealized || confirmed.Event.ResolvedAt == nil {
		t.Fatalf("confirmed = %+v", confirmed)
	}
	_, err = repo.Apply("lisa", model.TopicMatchDecision{
		Decision: model.TopicDecisionReuseEvent,
		Relation: model.EventRelationDisconfirmation,
		TopicID:  created.Topic.ID,
		EventID:  created.Event.ID,
		Reason:   "重複結果。",
	})
	if err == nil {
		t.Fatal("second resolution succeeded, want non-pending error")
	}
}

func TestTopicEventRepositoryProtectsAll32PendingEvents(t *testing.T) {
	repo := NewInMemoryTopicEventRepository()
	first, err := repo.Apply("lisa", newTopicDecision("事件-01", model.EventStatusPending))
	if err != nil {
		t.Fatal(err)
	}
	for i := 2; i <= TopicEventPoolCapacity; i++ {
		_, err := repo.Apply("lisa", newEventDecision(first.Topic.ID, fmt.Sprintf("事件-%02d", i), model.EventStatusPending))
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	_, err = repo.Apply("lisa", newEventDecision(first.Topic.ID, "事件-33", model.EventStatusActive))
	if !errors.Is(err, ErrEventPoolFull) {
		t.Fatalf("error = %v, want ErrEventPoolFull", err)
	}
	snapshot, err := repo.Candidates("lisa", TopicEventPoolCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Events) != TopicEventPoolCapacity {
		t.Fatalf("events = %d, want 32", len(snapshot.Events))
	}
	for _, event := range snapshot.Events {
		if event.Status != model.EventStatusPending {
			t.Fatalf("protected event was replaced: %+v", event)
		}
	}
}

func TestTopicEventRepositoryEvictsCompletedBeforeActiveAndNeverReusesID(t *testing.T) {
	repo := NewInMemoryTopicEventRepository()
	now := time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
	repo.now = func() time.Time {
		now = now.Add(time.Second)
		return now
	}
	first, err := repo.Apply("lisa", newTopicDecision("最舊已結束事件", model.EventStatusClosed))
	if err != nil {
		t.Fatal(err)
	}
	for i := 2; i <= TopicEventPoolCapacity; i++ {
		_, err := repo.Apply("lisa", newEventDecision(first.Topic.ID, fmt.Sprintf("活躍事件-%02d", i), model.EventStatusActive))
		if err != nil {
			t.Fatal(err)
		}
	}
	newest, err := repo.Apply("lisa", newEventDecision(first.Topic.ID, "新事件", model.EventStatusActive))
	if err != nil {
		t.Fatal(err)
	}
	if newest.Event.ID != "E-000033" {
		t.Fatalf("new Event ID = %q, want E-000033", newest.Event.ID)
	}
	snapshot, err := repo.Candidates("lisa", TopicEventPoolCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Events) != TopicEventPoolCapacity {
		t.Fatalf("events = %d, want 32", len(snapshot.Events))
	}
	for _, event := range snapshot.Events {
		if event.ID == first.Event.ID {
			t.Fatalf("completed event %s was not evicted", first.Event.ID)
		}
	}
}

func TestTopicEventRepositoryCandidateLimitAndIsolation(t *testing.T) {
	repo := NewInMemoryTopicEventRepository()
	_, err := repo.Apply("lisa", newTopicDecision("Lisa 事件", model.EventStatusActive))
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.Apply("william", newTopicDecision("William 事件", model.EventStatusActive))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := repo.Candidates("lisa", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Events) != 1 || snapshot.Events[0].CharacterID != "lisa" {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if _, err := repo.Candidates("lisa", TopicEventPoolCapacity+1); err == nil {
		t.Fatal("candidate limit 33 succeeded")
	}
}

func TestTopicEventRepositoryRejectsStaleProposalVersion(t *testing.T) {
	repo := NewInMemoryTopicEventRepository()
	snapshot, err := repo.Candidates("lisa", TopicEventPoolCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Version != 0 {
		t.Fatalf("initial version = %d, want 0", snapshot.Version)
	}
	_, err = repo.ApplyIfVersion("lisa", snapshot.Version, newTopicDecision("第一事件", model.EventStatusActive))
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.ApplyIfVersion("lisa", snapshot.Version, newTopicDecision("過期提案", model.EventStatusActive))
	if !errors.Is(err, ErrTopicPoolConflict) {
		t.Fatalf("error = %v, want ErrTopicPoolConflict", err)
	}
	current, err := repo.Candidates("lisa", TopicEventPoolCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if current.Version != 1 || len(current.Events) != 1 {
		t.Fatalf("snapshot after conflict = %+v", current)
	}
}

func newTopicDecision(summary string, status model.EventStatus) model.TopicMatchDecision {
	return model.TopicMatchDecision{
		Decision:     model.TopicDecisionNewTopicEvent,
		Relation:     model.EventRelationNew,
		TopicSummary: "玩家與 Lisa 的故事",
		EventSummary: summary,
		EventType:    "test",
		Subject:      "玩家",
		Target:       "Lisa",
		Status:       status,
		Reason:       "測試建立事件。",
	}
}

func newEventDecision(topicID, summary string, status model.EventStatus) model.TopicMatchDecision {
	decision := newTopicDecision(summary, status)
	decision.Decision = model.TopicDecisionNewEvent
	decision.TopicID = topicID
	decision.TopicSummary = ""
	return decision
}
