package affect

import (
	"context"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// BasicAppraisalSender submits one already interpreted Basic appraisal signal
// to an affect backend. ALMA implements it through POST /appraisal.
type BasicAppraisalSender interface {
	SendAppraisal(context.Context, model.BasicAppraisalSignal) (model.BasicAppraisalReceipt, error)
}
