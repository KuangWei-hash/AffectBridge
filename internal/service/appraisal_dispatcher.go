package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/KuangWei-hash/AffectBridge/internal/affect"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// BasicAppraisalDispatcher validates a complete 18-worker analysis and sends
// only matched tags to ALMA in canonical order.
type BasicAppraisalDispatcher struct {
	sender affect.BasicAppraisalSender
}

func NewBasicAppraisalDispatcher(sender affect.BasicAppraisalSender) *BasicAppraisalDispatcher {
	return &BasicAppraisalDispatcher{sender: sender}
}

type AppraisalDispatchReport struct {
	Signals  []model.BasicAppraisalSignal  `json:"signals"`
	Receipts []model.BasicAppraisalReceipt `json:"receipts"`
}

type AppraisalDispatchError struct {
	Signal       model.BasicAppraisalSignal
	AppliedCount int
	Err          error
}

func (e *AppraisalDispatchError) Error() string {
	return fmt.Sprintf("appraisal dispatch: %s failed after %d applied signals: %v",
		e.Signal.Tag, e.AppliedCount, e.Err)
}

func (e *AppraisalDispatchError) Unwrap() error { return e.Err }

// Dispatch performs all local validation before the first external write.
// ALMA exposes one-tag-at-a-time POST /appraisal, so a later transport failure
// cannot roll back earlier accepted signals; the returned report records every
// receipt obtained before failure.
func (d *BasicAppraisalDispatcher) Dispatch(
	ctx context.Context,
	characterName string,
	elicitor string,
	analysis model.AppraisalAnalysis,
) (AppraisalDispatchReport, error) {
	if d == nil || d.sender == nil {
		return AppraisalDispatchReport{}, errors.New("appraisal dispatch: sender is nil")
	}
	signals, err := compileBasicAppraisalSignals(characterName, elicitor, analysis)
	if err != nil {
		return AppraisalDispatchReport{}, err
	}
	report := AppraisalDispatchReport{
		Signals:  signals,
		Receipts: make([]model.BasicAppraisalReceipt, 0, len(signals)),
	}
	for _, signal := range signals {
		if err := ctx.Err(); err != nil {
			return report, &AppraisalDispatchError{Signal: signal, AppliedCount: len(report.Receipts), Err: err}
		}
		receipt, err := d.sender.SendAppraisal(ctx, signal)
		if err != nil {
			return report, &AppraisalDispatchError{Signal: signal, AppliedCount: len(report.Receipts), Err: err}
		}
		report.Receipts = append(report.Receipts, receipt)
	}
	return report, nil
}

func compileBasicAppraisalSignals(
	characterName string,
	elicitor string,
	analysis model.AppraisalAnalysis,
) ([]model.BasicAppraisalSignal, error) {
	characterName = strings.TrimSpace(characterName)
	if characterName == "" {
		return nil, errors.New("appraisal dispatch: character name is empty")
	}
	elicitor = strings.TrimSpace(elicitor)
	if elicitor == "" {
		return nil, errors.New("appraisal dispatch: elicitor is empty")
	}
	if len(analysis.Results) != model.AppraisalTagCount {
		return nil, fmt.Errorf("appraisal dispatch: got %d results, want %d", len(analysis.Results), model.AppraisalTagCount)
	}
	signals := make([]model.BasicAppraisalSignal, 0, model.AppraisalTagCount)
	for i, expectedTag := range model.AllAppraisalTags {
		result := analysis.Results[i]
		if err := result.Validate(expectedTag); err != nil {
			return nil, fmt.Errorf("appraisal dispatch: result %d: %w", i, err)
		}
		if !result.Matched {
			continue
		}
		intensity, err := appraisalIntensityValue(result.Intensity)
		if err != nil {
			return nil, err
		}
		signals = append(signals, model.BasicAppraisalSignal{
			Character: characterName,
			Tag:       result.Tag,
			Intensity: intensity,
			Elicitor:  elicitor,
		})
	}
	return signals, nil
}

func appraisalIntensityValue(intensity model.AppraisalIntensity) (float64, error) {
	switch intensity {
	case model.IntensityExtremeLight:
		return 0.10, nil
	case model.IntensityLight:
		return 0.25, nil
	case model.IntensityModerate:
		return 0.45, nil
	case model.IntensityStrong:
		return 0.65, nil
	case model.IntensityVeryStrong:
		return 0.85, nil
	case model.IntensityExtreme:
		return 1.00, nil
	default:
		return 0, fmt.Errorf("appraisal dispatch: intensity %q has no numeric mapping", intensity)
	}
}
