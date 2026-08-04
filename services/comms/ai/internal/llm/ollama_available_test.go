package llm_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/ai/internal/llm"
)

func TestOllamaAvailableTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2"}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("ERA_OLLAMA_URL", srv.URL)
	o := llm.NewOllamaFromEnv()
	if !o.Available() {
		t.Fatal("want Available() true against lab Ollama stub")
	}
	c := llm.FromEnv()
	if c.ModelName() == "heuristic" {
		t.Fatal("FromEnv should prefer Ollama when Available")
	}
}
