package model

// DialogueRole identifies who produced one dialogue message.
type DialogueRole string

const (
	DialogueRolePlayer    DialogueRole = "player"
	DialogueRoleCharacter DialogueRole = "character"
)

// DialogueMessage is one chronological player or character utterance.
// Dialogue history is kept separate from Character so it can later move to a
// dedicated conversation store without changing the affect domain model.
type DialogueMessage struct {
	Role    DialogueRole `json:"role"`
	Content string       `json:"content"`
}

// Story is the bounded narrative representation consumed by later appraisal
// stages. MaxOutputTokens records the generation budget requested from the LLM;
// it is not an estimate made from the returned string.
type Story struct {
	Text            string `json:"text"`
	SourceMessages  int    `json:"source_messages"`
	MaxOutputTokens int    `json:"max_output_tokens"`
}
