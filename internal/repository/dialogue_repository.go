package repository

import (
	"errors"
	"fmt"
	"sync"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

var ErrInvalidDialogueLimit = errors.New("dialogue repository: limit must be positive")

// DialogueRepository stores chronological utterances independently from the
// character affect model.
type DialogueRepository interface {
	Append(characterID string, messages ...model.DialogueMessage) error
	Recent(characterID string, limit int) ([]model.DialogueMessage, error)
}

// InMemoryDialogueRepository retains a bounded message history per character.
// It is process-local and intentionally follows the repository's existing
// experimental in-memory storage model.
type InMemoryDialogueRepository struct {
	mu          sync.RWMutex
	maxMessages int
	data        map[string][]model.DialogueMessage
}

func NewInMemoryDialogueRepository(maxMessages int) *InMemoryDialogueRepository {
	if maxMessages < 1 {
		maxMessages = 1
	}
	return &InMemoryDialogueRepository{
		maxMessages: maxMessages,
		data:        make(map[string][]model.DialogueMessage),
	}
}

func (r *InMemoryDialogueRepository) Append(
	characterID string,
	messages ...model.DialogueMessage,
) error {
	if characterID == "" {
		return errors.New("dialogue repository: character ID is empty")
	}
	for i, message := range messages {
		if message.Role != model.DialogueRolePlayer && message.Role != model.DialogueRoleCharacter {
			return fmt.Errorf("dialogue repository: messages[%d] has invalid role %q", i, message.Role)
		}
		if message.Content == "" {
			return fmt.Errorf("dialogue repository: messages[%d] content is empty", i)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	history := append(r.data[characterID], messages...)
	if len(history) > r.maxMessages {
		history = history[len(history)-r.maxMessages:]
	}
	r.data[characterID] = append([]model.DialogueMessage(nil), history...)
	return nil
}

func (r *InMemoryDialogueRepository) Recent(
	characterID string,
	limit int,
) ([]model.DialogueMessage, error) {
	if limit < 1 {
		return nil, ErrInvalidDialogueLimit
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	history := r.data[characterID]
	if len(history) > limit {
		history = history[len(history)-limit:]
	}
	return append([]model.DialogueMessage(nil), history...), nil
}
