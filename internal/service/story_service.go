package service

import (
	"context"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// StoryService is the first stage of the structured appraisal pipeline:
// recent dialogue -> bounded factual Story.
type StoryService struct {
	llm llm.Client
}

func NewStoryService(llmClient llm.Client) *StoryService {
	return &StoryService{llm: llmClient}
}

func (s *StoryService) Build(
	ctx context.Context,
	characterName string,
	dialogue []model.DialogueMessage,
) (model.Story, error) {
	return llm.BuildStory(ctx, s.llm, characterName, dialogue)
}
