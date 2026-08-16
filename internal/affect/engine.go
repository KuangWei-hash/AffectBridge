// Package affect defines the abstraction over the affective backend.
//
// AffectBridge uses the affect engine as the single source of truth
// for persistent character state. The first concrete backend is ALMA
// (see internal/affect/alma); other backends can be added without
// touching the rest of the system.
package affect

import (
	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// Engine is the interface every affective backend must satisfy.
type Engine interface {
	// Apply feeds an appraisal into the engine and returns the
	// updated character. The engine mutates mood and emotion (and
	// may adjust personality over very long timescales).
	Apply(current model.Character, appraisal model.Appraisal) (model.Character, error)

	// Snapshot returns the current affective state. It is read-only
	// and must not advance time or decay emotions.
	Snapshot(c model.Character) model.Character
}

// NewNoopEngine returns an engine that does not modify state.
// It is used until a real ALMA backend is wired in.
func NewNoopEngine() Engine {
	return &noopEngine{}
}

type noopEngine struct{}

func (n *noopEngine) Apply(c model.Character, _ model.Appraisal) (model.Character, error) {
	return c, nil
}

func (n *noopEngine) Snapshot(c model.Character) model.Character {
	return c
}
