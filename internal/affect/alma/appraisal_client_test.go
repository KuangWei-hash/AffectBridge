package alma

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KuangWei-hash/AffectBridge/internal/model"
)

func TestClientSendAppraisalPostsExactRESTPayload(t *testing.T) {
	want := model.BasicAppraisalSignal{
		Character: "Lisa",
		Tag:       model.GoodEvent,
		Intensity: 0.85,
		Elicitor:  "E-000101",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/appraisal" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		var got model.BasicAppraisalSignal
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if got != want {
			t.Errorf("signal = %+v, want %+v", got, want)
		}
		_ = json.NewEncoder(w).Encode(model.BasicAppraisalReceipt{
			Accepted:   true,
			Character:  got.Character,
			Tag:        got.Tag,
			Intensity:  got.Intensity,
			Elicitor:   got.Elicitor,
			SignalKind: "event",
		})
	}))
	defer server.Close()
	client := NewClient(server.URL + "/")
	client.httpc = server.Client()

	receipt, err := client.SendAppraisal(context.Background(), want)
	if err != nil {
		t.Fatalf("SendAppraisal() error = %v", err)
	}
	if !receipt.Accepted || receipt.Tag != model.GoodEvent || receipt.Elicitor != want.Elicitor {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestClientSendAppraisalReportsRESTErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"unknown character"}`, http.StatusNotFound)
	}))
	defer server.Close()
	client := NewClient(server.URL)
	client.httpc = server.Client()
	_, err := client.SendAppraisal(context.Background(), model.BasicAppraisalSignal{
		Character: "Lisa", Tag: model.GoodEvent, Intensity: 0.5, Elicitor: "E-1",
	})
	if err == nil || !strings.Contains(err.Error(), "status 404") || !strings.Contains(err.Error(), "unknown character") {
		t.Fatalf("error = %v", err)
	}
}

func TestClientSendAppraisalHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := NewClient("http://127.0.0.1:1")
	_, err := client.SendAppraisal(ctx, model.BasicAppraisalSignal{
		Character: "Lisa", Tag: model.GoodEvent, Intensity: 0.5, Elicitor: "E-1",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
