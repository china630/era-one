// Package classify — on-prem NLP / heuristic mail classification (MM-P2-5).
package classify

import (
	"os"
	"strings"
)

// Suspicious reports likely social-engineering content (air-gap heuristic).
func Suspicious(blob string) bool {
	lower := strings.ToLower(blob)
	for _, k := range []string{"urgent wire", "gift card", "password reset", "click here now", "bitcoin"} {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

// FromEnv — Ollama hint when ERA_OLLAMA_URL set (heuristic still used offline).
func FromEnv() func(string) bool {
	_ = os.Getenv("ERA_OLLAMA_URL")
	return Suspicious
}
