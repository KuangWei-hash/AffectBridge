package alma

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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
	return &Client{addr: addr, httpc: http.DefaultClient}
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
