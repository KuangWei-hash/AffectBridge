package model

// Personality captures long-term Big Five traits in [0, 1].
// These values are stable and rarely change during a session.
//
// In ALMA, personality defines the affective baseline upon which
// mood and emotion operate.
type Personality struct {
	Openness          float64 `json:"openness"`
	Conscientiousness float64 `json:"conscientiousness"`
	Extraversion      float64 `json:"extraversion"`
	Agreeableness     float64 `json:"agreeableness"`
	Neuroticism       float64 `json:"neuroticism"`
}

// Clamp returns a copy of the personality with each trait constrained
// to [0, 1]. It is called on input to keep untrusted values out of
// the affect engine.
func (p Personality) Clamp() Personality {
	return Personality{
		Openness:          clamp01(p.Openness),
		Conscientiousness: clamp01(p.Conscientiousness),
		Extraversion:      clamp01(p.Extraversion),
		Agreeableness:     clamp01(p.Agreeableness),
		Neuroticism:       clamp01(p.Neuroticism),
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
