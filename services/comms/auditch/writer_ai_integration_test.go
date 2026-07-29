//go:build integration

package auditch

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestAIInferenceAuditCH(t *testing.T) {
	addr := os.Getenv("ERA_CH_ADDR")
	if addr == "" {
		t.Skip("ERA_CH_ADDR not set")
	}
	w, err := New(addr)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := w.ApplyDDL(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.RecordAIInference(ctx, "t-demo", "mb-1", "phishing", "rule-based", 85, 12, "req-1", "deadbeef"); err != nil {
		t.Fatal(err)
	}
	n, err := w.CountAIInference(ctx, "phishing")
	if err != nil || n < 1 {
		t.Fatalf("ai inference rows=%d err=%v", n, err)
	}
}
