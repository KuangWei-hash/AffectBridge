package model

// BasicAppraisalSignal is the exact semantic payload sent to ALMA's
// POST /appraisal endpoint after one worker result has been validated.
type BasicAppraisalSignal struct {
	Character string       `json:"character"`
	Tag       AppraisalTag `json:"tag"`
	Intensity float64      `json:"intensity"`
	Elicitor  string       `json:"elicitor"`
}

// BasicAppraisalReceipt is ALMA's acknowledgement for one accepted signal.
type BasicAppraisalReceipt struct {
	Accepted   bool         `json:"accepted"`
	Character  string       `json:"character"`
	Tag        AppraisalTag `json:"tag"`
	Intensity  float64      `json:"intensity"`
	Elicitor   string       `json:"elicitor"`
	SignalKind string       `json:"signal_kind"`
}
