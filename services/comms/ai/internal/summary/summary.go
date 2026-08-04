package summary

import (
	"context"
	"fmt"
	"strings"
	"time"

	"era/services/comms/ai/internal/llm"
)

type Message struct {
	From    string `json:"from"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type Result struct {
	Summary   string `json:"summary"`
	Model     string `json:"model"`
	LatencyMs int64  `json:"latency_ms"`
}

func Summarize(ctx context.Context, client llm.Client, thread []Message) (Result, error) {
	if client == nil {
		client = llm.Heuristic{}
	}
	start := time.Now()
	prompt := buildPrompt(thread)
	text, err := client.Complete(ctx, prompt)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Summary:   strings.TrimSpace(text),
		Model:     client.ModelName(),
		LatencyMs: time.Since(start).Milliseconds(),
	}, nil
}

func buildPrompt(thread []Message) string {
	var b strings.Builder
	b.WriteString("Summarize the following email thread concisely:\n")
	for i, m := range thread {
		fmt.Fprintf(&b, "\n--- Message %d ---\nFrom: %s\nSubject: %s\n%s\n", i+1, m.From, m.Subject, m.Body)
	}
	return b.String()
}
