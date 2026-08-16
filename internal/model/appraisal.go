package model

// Appraisal is the structured, machine-readable description of an
// event that the LLM produces and the affect engine consumes.
//
// Field ranges follow ALMA's appraisal dimensions:
//   - Agency: "self" | "other" | "none"
//   - Desirability: [-1, 1]
//   - Unexpectedness: [0, 1]
//   - Blameworthiness: [0, 1]
//   - Praiseworthiness: [0, 1]
type Appraisal struct {
	Event            string  `json:"event"`
	Agency           string  `json:"agency"`
	Desirability     float64 `json:"desirability"`
	Unexpectedness   float64 `json:"unexpectedness"`
	Blameworthiness  float64 `json:"blameworthiness"`
	Praiseworthiness float64 `json:"praiseworthiness"`
}
