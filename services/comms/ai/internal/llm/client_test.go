package llm_test

import (
	"testing"

	"era/services/comms/ai/internal/llm"
)

func TestFromEnvFallsBackToHeuristic(t *testing.T) {
	t.Setenv("ERA_OLLAMA_URL", "http://127.0.0.1:1")
	c := llm.FromEnv()
	if c.ModelName() != "heuristic" {
		t.Fatalf("want heuristic without ollama, got %s", c.ModelName())
	}
	if !c.Available() {
		t.Fatal("heuristic must be available")
	}
}
