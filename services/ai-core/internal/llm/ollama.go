// Package llm — on-prem inference adapters (GA-1 S5-13).
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Client interface {
	Complete(ctx context.Context, prompt string) (string, error)
	Available() bool
}

type Ollama struct {
	BaseURL string
	Model   string
	client  *http.Client
}

func NewOllamaFromEnv() *Ollama {
	url := os.Getenv("ERA_OLLAMA_URL")
	if url == "" {
		url = "http://127.0.0.1:11434"
	}
	model := os.Getenv("ERA_OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2"
	}
	return &Ollama{BaseURL: url, Model: model, client: &http.Client{Timeout: 60 * time.Second}}
}

func (o *Ollama) Available() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, o.BaseURL+"/api/tags", nil)
	resp, err := o.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (o *Ollama) Complete(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":  o.Model,
		"prompt": prompt,
		"stream": false,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama status %d", resp.StatusCode)
	}
	var out struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Response, nil
}

type Heuristic struct{}

func (Heuristic) Available() bool { return true }

func (Heuristic) Complete(_ context.Context, prompt string) (string, error) {
	return "heuristic: " + prompt[:min(120, len(prompt))], nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func FromEnv() Client {
	o := NewOllamaFromEnv()
	if o.Available() {
		return o
	}
	return Heuristic{}
}
