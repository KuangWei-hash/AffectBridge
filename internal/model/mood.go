package model

// Mood is the medium-term PAD (Pleasure-Arousal-Dominance) state.
// Each axis is in [-1, 1] following ALMA's convention.
//
// Mood is slower than emotion and faster than personality. It can
// persist across many events and survive short emotional spikes.
type Mood struct {
	Pleasure  float64 `json:"pleasure"`
	Arousal   float64 `json:"arousal"`
	Dominance float64 `json:"dominance"`
}
