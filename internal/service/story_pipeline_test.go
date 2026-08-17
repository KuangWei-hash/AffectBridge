package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
)

type pipelineStoryStub struct {
	prompt string
}

func (s *pipelineStoryStub) Complete(_ context.Context, prompt string, _ ...llm.Option) (string, error) {
	s.prompt = prompt
	return "最近對話的故事", nil
}

func TestStoryPipelineBuildsFromNineCommittedMessagesPlusCurrentPlayer(t *testing.T) {
	history := repository.NewInMemoryDialogueRepository(llm.StoryDialogueWindow)
	for i := 1; i <= llm.StoryDialogueWindow; i++ {
		role := model.DialogueRolePlayer
		if i%2 == 0 {
			role = model.DialogueRoleCharacter
		}
		if err := history.Append("lisa", model.DialogueMessage{
			Role:    role,
			Content: fmt.Sprintf("old-%02d", i),
		}); err != nil {
			t.Fatal(err)
		}
	}

	stub := &pipelineStoryStub{}
	pipeline := NewStoryPipeline(NewStoryService(stub), history)
	got, err := pipeline.BuildForPlayerMessage(context.Background(), "lisa", "Lisa", "current")
	if err != nil {
		t.Fatalf("BuildForPlayerMessage() error = %v", err)
	}
	if got.SourceMessages != llm.StoryDialogueWindow {
		t.Fatalf("SourceMessages = %d, want %d", got.SourceMessages, llm.StoryDialogueWindow)
	}

	var payload struct {
		Dialogue []model.DialogueMessage `json:"dialogue"`
	}
	if err := json.Unmarshal([]byte(stub.prompt), &payload); err != nil {
		t.Fatalf("decode story prompt: %v", err)
	}
	if payload.Dialogue[0].Content != "old-02" || payload.Dialogue[9].Content != "current" {
		t.Fatalf("window = %q ... %q, want old-02 ... current",
			payload.Dialogue[0].Content, payload.Dialogue[9].Content)
	}

	stored, err := history.Recent("lisa", llm.StoryDialogueWindow)
	if err != nil {
		t.Fatal(err)
	}
	if stored[len(stored)-1].Content != "old-10" {
		t.Fatal("BuildForPlayerMessage() mutated committed history")
	}
}

func TestStoryPipelineCommitsCompletedExchange(t *testing.T) {
	history := repository.NewInMemoryDialogueRepository(llm.StoryDialogueWindow)
	pipeline := NewStoryPipeline(NewStoryService(&pipelineStoryStub{}), history)
	if err := pipeline.CommitExchange("lisa", "我會回來", "我會等你"); err != nil {
		t.Fatalf("CommitExchange() error = %v", err)
	}

	got, err := history.Recent("lisa", llm.StoryDialogueWindow)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Role != model.DialogueRolePlayer || got[1].Role != model.DialogueRoleCharacter {
		t.Fatalf("history = %+v", got)
	}
}
