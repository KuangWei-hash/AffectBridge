// Package service holds the business logic of AffectBridge.
//
// Services are the only place that mutates the affective state of a
// character. They depend on the repository and the affect engine,
// but not on the HTTP layer.
package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
)

type CharacterService struct {
	repo repository.CharacterRepository
}

func NewCharacterService(repo repository.CharacterRepository) *CharacterService {
	return &CharacterService{repo: repo}
}

// Create starts a new character with the given personality. Mood
// starts at the neutral PAD origin; emotions start empty.
func (s *CharacterService) Create(name string, p model.Personality) (*model.Character, error) {
	now := time.Now()
	c := &model.Character{
		ID:          newID(),
		Name:        name,
		Personality: p.Clamp(),
		Mood:        model.Mood{},
		Emotions:    model.EmotionSet{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CharacterService) Get(id string) (*model.Character, error) {
	return s.repo.Find(id)
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
