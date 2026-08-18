package service

import (
	"context"
	"errors"
	"testing"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

type appraisalSenderStub struct {
	signals []model.BasicAppraisalSignal
	failAt  int
	err     error
}

func (s *appraisalSenderStub) SendAppraisal(_ context.Context, signal model.BasicAppraisalSignal) (model.BasicAppraisalReceipt, error) {
	if s.failAt > 0 && len(s.signals)+1 == s.failAt {
		return model.BasicAppraisalReceipt{}, s.err
	}
	s.signals = append(s.signals, signal)
	return model.BasicAppraisalReceipt{
		Accepted: true, Character: signal.Character, Tag: signal.Tag,
		Intensity: signal.Intensity, Elicitor: signal.Elicitor, SignalKind: "event",
	}, nil
}

func TestBasicAppraisalDispatcherSendsOnlyMatchedTagsInCanonicalOrder(t *testing.T) {
	sender := &appraisalSenderStub{}
	dispatcher := NewBasicAppraisalDispatcher(sender)
	analysis := completeAnalysis(map[model.AppraisalTag]model.AppraisalIntensity{
		model.GoodEvent:      model.IntensityVeryStrong,
		model.EventConfirmed: model.IntensityStrong,
		model.GoodActOther:   model.IntensityModerate,
	})
	report, err := dispatcher.Dispatch(context.Background(), "Lisa", "E-000101", analysis)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if len(sender.signals) != 3 || len(report.Receipts) != 3 {
		t.Fatalf("signals=%d receipts=%d, want 3", len(sender.signals), len(report.Receipts))
	}
	wantTags := []model.AppraisalTag{model.GoodEvent, model.EventConfirmed, model.GoodActOther}
	wantIntensities := []float64{0.85, 0.65, 0.45}
	for i, signal := range sender.signals {
		if signal.Tag != wantTags[i] || signal.Intensity != wantIntensities[i] ||
			signal.Character != "Lisa" || signal.Elicitor != "E-000101" {
			t.Fatalf("signal[%d] = %+v", i, signal)
		}
	}
}

func TestBasicAppraisalDispatcherValidatesWholeBatchBeforeSending(t *testing.T) {
	sender := &appraisalSenderStub{}
	dispatcher := NewBasicAppraisalDispatcher(sender)
	_, err := dispatcher.Dispatch(context.Background(), "Lisa", "E-1", model.AppraisalAnalysis{
		Results: completeAnalysis(nil).Results[:17],
	})
	if err == nil {
		t.Fatal("Dispatch() succeeded with incomplete analysis")
	}
	if len(sender.signals) != 0 {
		t.Fatalf("sent %d signals before validation failed", len(sender.signals))
	}
}

func TestBasicAppraisalDispatcherReportsPartialExternalApplication(t *testing.T) {
	want := errors.New("ALMA unavailable")
	sender := &appraisalSenderStub{failAt: 2, err: want}
	dispatcher := NewBasicAppraisalDispatcher(sender)
	report, err := dispatcher.Dispatch(context.Background(), "Lisa", "E-1", completeAnalysis(map[model.AppraisalTag]model.AppraisalIntensity{
		model.GoodEvent: model.IntensityStrong,
		model.BadEvent:  model.IntensityLight,
	}))
	var dispatchErr *AppraisalDispatchError
	if !errors.As(err, &dispatchErr) || !errors.Is(err, want) {
		t.Fatalf("error = %v, want AppraisalDispatchError wrapping provider error", err)
	}
	if dispatchErr.AppliedCount != 1 || len(report.Receipts) != 1 || len(report.Signals) != 2 {
		t.Fatalf("error=%+v report=%+v", dispatchErr, report)
	}
}

func TestAppraisalIntensityMapping(t *testing.T) {
	tests := map[model.AppraisalIntensity]float64{
		model.IntensityExtremeLight: 0.10,
		model.IntensityLight:        0.25,
		model.IntensityModerate:     0.45,
		model.IntensityStrong:       0.65,
		model.IntensityVeryStrong:   0.85,
		model.IntensityExtreme:      1.00,
	}
	for intensity, want := range tests {
		got, err := appraisalIntensityValue(intensity)
		if err != nil || got != want {
			t.Fatalf("mapping %s = %v, %v; want %v", intensity, got, err, want)
		}
	}
}

func completeAnalysis(matched map[model.AppraisalTag]model.AppraisalIntensity) model.AppraisalAnalysis {
	results := make([]model.AppraisalWorkerResult, 0, model.AppraisalTagCount)
	for _, tag := range model.AllAppraisalTags {
		intensity, ok := matched[tag]
		result := model.AppraisalWorkerResult{
			Tag: tag, Matched: ok, Reason: "沒有足夠證據。", Intensity: model.IntensityNotApplicable,
		}
		if ok {
			result.Reason = "故事提供明確證據。"
			result.Intensity = intensity
		}
		results = append(results, result)
	}
	return model.AppraisalAnalysis{Results: results}
}
