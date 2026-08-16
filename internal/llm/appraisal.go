package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// Appraise asks the LLM to turn a natural-language event into a
// structured Appraisal. This corresponds to the "semantic layer"
// step in the AffectBridge architecture.
func Appraise(ctx context.Context, client Client, event string) (model.Appraisal, error) {
	const system = `You are a structured appraisal engine.
Given an event, return a JSON object with these fields:
- agency: "self" | "other" | "none"
- desirability: float in [-1, 1]
- unexpectedness: float in [0, 1]
- blameworthiness: float in [0, 1]
- praiseworthiness: float in [0, 1]
Return only the JSON object, no commentary.`

	raw, err := client.Complete(ctx, event, WithSystem(system))
	if err != nil {
		return model.Appraisal{}, err
	}

	var a model.Appraisal
	a.Event = event
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		return a, fmt.Errorf("appraisal: parse llm output: %w", err)
	}
	return a, nil
}
