package service

import (
	"context"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// AppraisalService turns a natural-language event into a structured
// Appraisal by asking the LLM. It is the "semantic layer" of the
// pipeline; the affect engine consumes its output.
type AppraisalService struct {
	llm llm.Client
}

func NewAppraisalService(llmClient llm.Client) *AppraisalService {
	return &AppraisalService{llm: llmClient}
}

func (s *AppraisalService) Appraise(ctx context.Context, event string) (model.Appraisal, error) {
	return llm.Appraise(ctx, s.llm, event)
}
