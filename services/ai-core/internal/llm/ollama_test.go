package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"response": "malicious lateral movement"})
	}))
	defer srv.Close()

	o := &Ollama{BaseURL: srv.URL, Model: "test", client: srv.Client()}
	if !o.Available() {
		t.Fatal("expected available")
	}
	text, err := o.Complete(context.Background(), "analyze incident")
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("empty response")
	}
}
