package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// ChatService is the top-level orchestration. It is the only place
// that knows the full pipeline:
//
//	player message
//	  -> appraisal (LLM)
//	  -> apply affect (engine)
//	  -> express current state (LLM)
//	  -> reply
type ChatService struct {
	llm       llm.Client
	appraisal *AppraisalService
	affect    *AffectService
}

func NewChatService(llmClient llm.Client, appraisal *AppraisalService, affect *AffectService) *ChatService {
	return &ChatService{llm: llmClient, appraisal: appraisal, affect: affect}
}

// ChatReply is the response a caller gets from POST /characters/{id}/chat.
type ChatReply struct {
	Character *model.Character `json:"character"`
	Appraisal model.Appraisal  `json:"appraisal"`
	Reply     string           `json:"reply"`
}

// Send runs the full pipeline for a single player message.
func (s *ChatService) Send(ctx context.Context, c *model.Character, message string) (*ChatReply, error) {
	appraisal, err := s.appraisal.Appraise(ctx, message)
	if err != nil {
		return nil, err
	}

	updated, err := s.affect.Apply(c, appraisal)
	if err != nil {
		return nil, err
	}

	const system = `You are the voice of a game character.
You are given the character's current mood and active emotions.
Express that state naturally in dialogue. Do not narrate the state;
do not break character.`

	reply, err := s.llm.Complete(ctx, buildPrompt(message, updated), llm.WithSystem(system))
	if err != nil {
		return nil, err
	}

	return &ChatReply{
		Character: updated,
		Appraisal: appraisal,
		Reply:     reply,
	}, nil
}

func buildPrompt(message string, c *model.Character) string {
	return fmt.Sprintf(
		"Character state:\n- Mood (P/A/D): %.2f / %.2f / %.2f\n- Active emotions: %s\n\nPlayer message: %s\n\nReply in character.",
		c.Mood.Pleasure, c.Mood.Arousal, c.Mood.Dominance,
		formatEmotions(c.Emotions),
		message,
	)
}

func formatEmotions(e model.EmotionSet) string {
	if len(e) == 0 {
		return "(none)"
	}
	pairs := make([]string, 0, len(e))
	for k, v := range e {
		pairs = append(pairs, fmt.Sprintf("%s=%.2f", k, v))
	}
	return strings.Join(pairs, ", ")
}
