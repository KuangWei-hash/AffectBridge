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
	const schema = `{
  "type": "object",
  "properties": {
    "agency": {"type": "string", "enum": ["self", "other", "none"]},
    "desirability": {"type": "number", "minimum": -1, "maximum": 1},
    "unexpectedness": {"type": "number", "minimum": 0, "maximum": 1},
    "blameworthiness": {"type": "number", "minimum": 0, "maximum": 1},
    "praiseworthiness": {"type": "number", "minimum": 0, "maximum": 1}
  },
  "required": ["agency", "desirability", "unexpectedness", "blameworthiness", "praiseworthiness"],
  "additionalProperties": false
}`

	// The configured reasoning effort applies at the client level. A compact
	// budget is enough for this small structured result when reasoning is off.
	raw, err := client.Complete(ctx, event,
		WithSystem(system),
		WithJSONSchema(schema),
		WithMaxTokens(512),
	)
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
