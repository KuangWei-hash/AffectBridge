package repository

import (
	"errors"
	"testing"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

func TestInMemoryDialogueRepositoryRetainsBoundedRecentMessages(t *testing.T) {
	repo := NewInMemoryDialogueRepository(3)
	for i, content := range []string{"one", "two", "three", "four"} {
		role := model.DialogueRolePlayer
		if i%2 == 1 {
			role = model.DialogueRoleCharacter
		}
		if err := repo.Append("lisa", model.DialogueMessage{Role: role, Content: content}); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	got, err := repo.Recent("lisa", 10)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(got) != 3 || got[0].Content != "two" || got[2].Content != "four" {
		t.Fatalf("Recent() = %+v, want two..four", got)
	}

	got[0].Content = "mutated"
	again, err := repo.Recent("lisa", 10)
	if err != nil {
		t.Fatalf("Recent() second call error = %v", err)
	}
	if again[0].Content != "two" {
		t.Fatal("Recent() returned storage owned by repository")
	}
}

func TestInMemoryDialogueRepositorySeparatesCharacters(t *testing.T) {
	repo := NewInMemoryDialogueRepository(10)
	if err := repo.Append("lisa", model.DialogueMessage{Role: model.DialogueRolePlayer, Content: "Lisa"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Append("william", model.DialogueMessage{Role: model.DialogueRolePlayer, Content: "William"}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Recent("lisa", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Content != "Lisa" {
		t.Fatalf("Lisa history = %+v", got)
	}
}

func TestInMemoryDialogueRepositoryRejectsInvalidLimit(t *testing.T) {
	repo := NewInMemoryDialogueRepository(10)
	_, err := repo.Recent("lisa", 0)
	if !errors.Is(err, ErrInvalidDialogueLimit) {
		t.Fatalf("error = %v, want ErrInvalidDialogueLimit", err)
	}
}
