package service

import (
	"github.com/KuangWei-hash/AffectBridge/internal/affect"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
)

type AffectService struct {
	engine affect.Engine
	repo   repository.CharacterRepository
}

func NewAffectService(engine affect.Engine, repo repository.CharacterRepository) *AffectService {
	return &AffectService{engine: engine, repo: repo}
}

// Apply feeds an appraisal into the affect engine and persists the
// updated character. The returned character reflects the new mood
// and emotion intensities after the engine has digested the
// appraisal.
func (s *AffectService) Apply(c *model.Character, appraisal model.Appraisal) (*model.Character, error) {
	updated, err := s.engine.Apply(*c, appraisal)
	if err != nil {
		return c, err
	}
	if err := s.repo.Save(&updated); err != nil {
		return c, err
	}
	return &updated, nil
}

// Snapshot returns the engine's view of the current state. It does
// not advance time or mutate anything.
func (s *AffectService) Snapshot(c *model.Character) model.Character {
	return s.engine.Snapshot(*c)
}
