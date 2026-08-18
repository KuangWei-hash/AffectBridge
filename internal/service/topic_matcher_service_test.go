package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
)

type serializedTopicClient struct {
	mu             sync.Mutex
	emptyCalls     int
	nonEmptyCalls  int
	firstStarted   chan struct{}
	releaseFirst   chan struct{}
	firstStartOnce sync.Once
}

func newSerializedTopicClient() *serializedTopicClient {
	return &serializedTopicClient{
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan struct{}),
	}
}

func (c *serializedTopicClient) Complete(ctx context.Context, prompt string, _ ...llm.Option) (string, error) {
	var input struct {
		Events []model.NarrativeEvent `json:"events"`
	}
	if err := json.Unmarshal([]byte(prompt), &input); err != nil {
		return "", err
	}
	if len(input.Events) == 0 {
		c.mu.Lock()
		c.emptyCalls++
		c.mu.Unlock()
		c.firstStartOnce.Do(func() { close(c.firstStarted) })
		select {
		case <-c.releaseFirst:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		return `{
          "decision":"new_topic_event","relation":"new",
          "topic_id":"","event_id":"",
          "topic_summary":"玩家與 Lisa 的承諾",
          "event_summary":"玩家承諾返回 Lisa 身邊","event_type":"promise",
          "subject":"玩家","target":"Lisa","status":"pending",
          "reason":"沒有既有候選。"
        }`, nil
	}
	c.mu.Lock()
	c.nonEmptyCalls++
	c.mu.Unlock()
	event := input.Events[0]
	return fmt.Sprintf(`{
      "decision":"reuse_event","relation":"continuation",
      "topic_id":%q,"event_id":%q,
      "topic_summary":"","event_summary":"","event_type":"",
      "subject":"","target":"","status":"",
      "reason":"故事延續同一項承諾。"
    }`, event.TopicID, event.ID), nil
}

func TestTopicMatcherServiceSerializesSameCharacterResolution(t *testing.T) {
	client := newSerializedTopicClient()
	repo := repository.NewInMemoryTopicEventRepository()
	service := NewTopicMatcherService(client, repo)
	type response struct {
		resolution model.TopicResolution
		err        error
	}
	responses := make(chan response, 2)

	go func() {
		resolution, err := service.Resolve(context.Background(), "lisa", "Lisa", model.Story{Text: "玩家承諾返回。"})
		responses <- response{resolution: resolution, err: err}
	}()
	<-client.firstStarted
	go func() {
		resolution, err := service.Resolve(context.Background(), "lisa", "Lisa", model.Story{Text: "玩家再次談及同一承諾。"})
		responses <- response{resolution: resolution, err: err}
	}()
	close(client.releaseFirst)

	first := <-responses
	second := <-responses
	if first.err != nil || second.err != nil {
		t.Fatalf("errors = %v, %v", first.err, second.err)
	}
	if first.resolution.Event.ID != second.resolution.Event.ID {
		t.Fatalf("Event IDs = %q and %q, want reuse", first.resolution.Event.ID, second.resolution.Event.ID)
	}
	client.mu.Lock()
	emptyCalls, nonEmptyCalls := client.emptyCalls, client.nonEmptyCalls
	client.mu.Unlock()
	if emptyCalls != 1 || nonEmptyCalls != 1 {
		t.Fatalf("empty calls=%d non-empty calls=%d, want 1 and 1", emptyCalls, nonEmptyCalls)
	}
	snapshot, err := repo.Candidates("lisa", repository.TopicEventPoolCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Events) != 1 {
		t.Fatalf("pool events = %d, want 1", len(snapshot.Events))
	}
}

func TestTopicMatcherServiceCancellationDoesNotMutatePool(t *testing.T) {
	client := newSerializedTopicClient()
	repo := repository.NewInMemoryTopicEventRepository()
	service := NewTopicMatcherService(client, repo)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.Resolve(ctx, "lisa", "Lisa", model.Story{Text: "故事"})
	if err == nil {
		t.Fatal("Resolve() succeeded with cancelled context")
	}
	snapshot, snapshotErr := repo.Candidates("lisa", repository.TopicEventPoolCapacity)
	if snapshotErr != nil {
		t.Fatal(snapshotErr)
	}
	if len(snapshot.Events) != 0 || len(snapshot.Topics) != 0 {
		t.Fatalf("cancelled resolution mutated pool: %+v", snapshot)
	}
}

func TestTopicMatcherServiceProposalDoesNotMutateUntilCommit(t *testing.T) {
	client := newSerializedTopicClient()
	close(client.releaseFirst)
	repo := repository.NewInMemoryTopicEventRepository()
	service := NewTopicMatcherService(client, repo)

	proposal, err := service.Propose(context.Background(), "lisa", "Lisa", model.Story{Text: "玩家承諾返回。"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := repo.Candidates("lisa", repository.TopicEventPoolCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Events) != 0 || before.Version != 0 {
		t.Fatalf("proposal mutated pool: %+v", before)
	}
	resolution, err := service.Commit(proposal)
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.Created || resolution.Event.ID == "" {
		t.Fatalf("resolution = %+v", resolution)
	}
	after, err := repo.Candidates("lisa", repository.TopicEventPoolCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Events) != 1 || after.Version != 1 {
		t.Fatalf("committed pool = %+v", after)
	}
}

func TestTopicMatcherServiceRejectsSecondProposalFromStaleSnapshot(t *testing.T) {
	client := newSerializedTopicClient()
	close(client.releaseFirst)
	repo := repository.NewInMemoryTopicEventRepository()
	service := NewTopicMatcherService(client, repo)
	first, err := service.Propose(context.Background(), "lisa", "Lisa", model.Story{Text: "事件一"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Propose(context.Background(), "lisa", "Lisa", model.Story{Text: "事件二"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Commit(first); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Commit(second); !errors.Is(err, repository.ErrTopicPoolConflict) {
		t.Fatalf("error = %v, want ErrTopicPoolConflict", err)
	}
}
