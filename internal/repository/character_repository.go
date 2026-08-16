// Package repository hides the persistence layer behind an interface.
// The default implementation is in-memory; swap in a real database
// by providing another CharacterRepository.
package repository

import (
	"errors"
	"sync"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// ErrNotFound is returned when a character ID does not exist.
var ErrNotFound = errors.New("character not found")

type CharacterRepository interface {
	Save(c *model.Character) error
	Find(id string) (*model.Character, error)
}

// InMemoryCharacterRepository is a process-local store. It is the
// intended default for the initial AffectBridge experiment; it is
// not durable across restarts.
type InMemoryCharacterRepository struct {
	mu   sync.RWMutex
	data map[string]*model.Character
}

func NewInMemoryCharacterRepository() *InMemoryCharacterRepository {
	return &InMemoryCharacterRepository{data: make(map[string]*model.Character)}
}

func (r *InMemoryCharacterRepository) Save(c *model.Character) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[c.ID] = c
	return nil
}

func (r *InMemoryCharacterRepository) Find(id string) (*model.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}
