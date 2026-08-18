package model

import "fmt"

// AppraisalTag identifies one Boolean appraisal dimension evaluated by the
// structured 18-worker pipeline. The order in AllAppraisalTags is canonical.
type AppraisalTag string

const (
	GoodEvent               AppraisalTag = "GoodEvent"
	BadEvent                AppraisalTag = "BadEvent"
	GoodEventForGoodOther   AppraisalTag = "GoodEventForGoodOther"
	GoodEventForBadOther    AppraisalTag = "GoodEventForBadOther"
	BadEventForGoodOther    AppraisalTag = "BadEventForGoodOther"
	BadEventForBadOther     AppraisalTag = "BadEventForBadOther"
	GoodLikelyFutureEvent   AppraisalTag = "GoodLikelyFutureEvent"
	GoodUnlikelyFutureEvent AppraisalTag = "GoodUnlikelyFutureEvent"
	BadLikelyFutureEvent    AppraisalTag = "BadLikelyFutureEvent"
	BadUnlikelyFutureEvent  AppraisalTag = "BadUnlikelyFutureEvent"
	EventConfirmed          AppraisalTag = "EventConfirmed"
	EventDisconfirmed       AppraisalTag = "EventDisconfirmed"
	GoodActSelf             AppraisalTag = "GoodActSelf"
	GoodActOther            AppraisalTag = "GoodActOther"
	BadActSelf              AppraisalTag = "BadActSelf"
	BadActOther             AppraisalTag = "BadActOther"
	NiceThing               AppraisalTag = "NiceThing"
	NastyThing              AppraisalTag = "NastyThing"
)

const AppraisalTagCount = 18

var AllAppraisalTags = [AppraisalTagCount]AppraisalTag{
	GoodEvent,
	BadEvent,
	GoodEventForGoodOther,
	GoodEventForBadOther,
	BadEventForGoodOther,
	BadEventForBadOther,
	GoodLikelyFutureEvent,
	GoodUnlikelyFutureEvent,
	BadLikelyFutureEvent,
	BadUnlikelyFutureEvent,
	EventConfirmed,
	EventDisconfirmed,
	GoodActSelf,
	GoodActOther,
	BadActSelf,
	BadActOther,
	NiceThing,
	NastyThing,
}

func (t AppraisalTag) Valid() bool {
	for _, candidate := range AllAppraisalTags {
		if t == candidate {
			return true
		}
	}
	return false
}

// AppraisalIntensity is the six-level semantic intensity scale shared by all
// workers. NotApplicable is mandatory whenever Matched is false.
type AppraisalIntensity string

const (
	IntensityExtremeLight  AppraisalIntensity = "極輕微"
	IntensityLight         AppraisalIntensity = "輕微"
	IntensityModerate      AppraisalIntensity = "中等"
	IntensityStrong        AppraisalIntensity = "強烈"
	IntensityVeryStrong    AppraisalIntensity = "非常強烈"
	IntensityExtreme       AppraisalIntensity = "極端"
	IntensityNotApplicable AppraisalIntensity = "不適用"
)

func (i AppraisalIntensity) Valid() bool {
	switch i {
	case IntensityExtremeLight, IntensityLight, IntensityModerate,
		IntensityStrong, IntensityVeryStrong, IntensityExtreme,
		IntensityNotApplicable:
		return true
	default:
		return false
	}
}

// AppraisalWorkerResult is the validated output of one specialized worker.
type AppraisalWorkerResult struct {
	Tag       AppraisalTag       `json:"tag"`
	Matched   bool               `json:"matched"`
	Reason    string             `json:"reason"`
	Intensity AppraisalIntensity `json:"intensity"`
}

func (r AppraisalWorkerResult) Validate(expected AppraisalTag) error {
	if !expected.Valid() {
		return fmt.Errorf("invalid expected appraisal tag %q", expected)
	}
	if r.Tag != expected {
		return fmt.Errorf("worker returned tag %q, want %q", r.Tag, expected)
	}
	if r.Reason == "" {
		return fmt.Errorf("worker %s returned an empty reason", expected)
	}
	if !r.Intensity.Valid() {
		return fmt.Errorf("worker %s returned invalid intensity %q", expected, r.Intensity)
	}
	if r.Matched && r.Intensity == IntensityNotApplicable {
		return fmt.Errorf("worker %s matched but intensity is not applicable", expected)
	}
	if !r.Matched && r.Intensity != IntensityNotApplicable {
		return fmt.Errorf("worker %s did not match but intensity is %q", expected, r.Intensity)
	}
	return nil
}

// AppraisalAnalysis contains successful worker results in canonical tag order.
// Individual worker failures may produce partial Results with an error that
// identifies every failed tag. Batch timeout or cancellation returns no Results.
type AppraisalAnalysis struct {
	Results []AppraisalWorkerResult `json:"results"`
}
