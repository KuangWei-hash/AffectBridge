// Package alma is the ALMA-backed implementation of affect.Engine.
//
// ALMA itself is a separate Java program. This package treats that
// runtime as an external dependency and speaks to it over HTTP.
// AffectBridge does not reimplement ALMA and does not redistribute it.
package alma

import (
	"github.com/KuangWei-hash/AffectBridge/internal/affect"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// Engine is an ALMA-backed affect engine.
type Engine struct {
	client *Client
}

func NewEngine(addr string) *Engine {
	return &Engine{client: NewClient(addr)}
}

func (e *Engine) Apply(c model.Character, appraisal model.Appraisal) (model.Character, error) {
	return e.client.Apply(c, appraisal)
}

func (e *Engine) Snapshot(c model.Character) model.Character {
	return e.client.Snapshot(c)
}

// Compile-time check that ALMA's Engine satisfies affect.Engine.
var _ affect.Engine = (*Engine)(nil)
