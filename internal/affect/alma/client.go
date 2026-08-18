package alma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

// Client is a thin HTTP client for a running ALMA runtime.
//
// The wire format is intentionally simple: ALMA receives a Character
// and an Appraisal, and returns an updated Character. The exact HTTP
// surface of ALMA is TBD; this package is the single place that
// knows about it.
type Client struct {
	addr  string
	httpc *http.Client
}

func NewClient(addr string) *Client {
	return &Client{addr: strings.TrimRight(addr, "/"), httpc: http.DefaultClient}
}

// SendAppraisal sends one validated Basic tag to ALMA's synchronous REST
// appraisal endpoint. The caller's context owns the complete request lifetime.
func (c *Client) SendAppraisal(
	ctx context.Context,
	signal model.BasicAppraisalSignal,
) (model.BasicAppraisalReceipt, error) {
	body, err := json.Marshal(signal)
	if err != nil {
		return model.BasicAppraisalReceipt{}, fmt.Errorf("alma: appraisal encode: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.addr+"/appraisal", bytes.NewReader(body))
	if err != nil {
		return model.BasicAppraisalReceipt{}, fmt.Errorf("alma: appraisal request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return model.BasicAppraisalReceipt{}, fmt.Errorf("alma: appraisal: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return model.BasicAppraisalReceipt{}, fmt.Errorf("alma: appraisal: status %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	var receipt model.BasicAppraisalReceipt
	if err := json.NewDecoder(resp.Body).Decode(&receipt); err != nil {
		return model.BasicAppraisalReceipt{}, fmt.Errorf("alma: appraisal decode: %w", err)
	}
	if !receipt.Accepted || receipt.Character != signal.Character || receipt.Tag != signal.Tag ||
		receipt.Elicitor != signal.Elicitor {
		return model.BasicAppraisalReceipt{}, fmt.Errorf("alma: appraisal acknowledgement does not match request")
	}
	return receipt, nil
}

type applyRequest struct {
	Character model.Character `json:"character"`
	Appraisal model.Appraisal `json:"appraisal"`
}

type applyResponse struct {
	Character model.Character `json:"character"`
}

func (c *Client) Apply(current model.Character, appraisal model.Appraisal) (model.Character, error) {
	body, err := json.Marshal(applyRequest{
		Character: current,
		Appraisal: appraisal,
	})
	if err != nil {
		return current, err
	}

	resp, err := c.httpc.Post(c.addr+"/apply", "application/json", bytes.NewReader(body))
	if err != nil {
		return current, fmt.Errorf("alma: apply: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return current, fmt.Errorf("alma: apply: status %d", resp.StatusCode)
	}

	var out applyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return current, err
	}
	return out.Character, nil
}

func (c *Client) Snapshot(c2 model.Character) model.Character {
	return c2
}
