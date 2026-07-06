// Package hub — federated learning hub с DP-агрегацией (ADR-0002, F3-4).
package hub

import (
	"math"
	"math/rand"
	"sync"
)

type GradientSubmission struct {
	ZoneID      string    `json:"zone_id"`
	Vector      []float64 `json:"vector"`
	SampleCount int       `json:"sample_count"`
}

type Hub struct {
	mu          sync.Mutex
	epsilon     float64
	submissions map[string][]GradientSubmission
	global      []float64
	round       int
	audit       *AuditLog
}

func New(epsilon float64) *Hub {
	if epsilon <= 0 {
		epsilon = 1.0
	}
	return &Hub{epsilon: epsilon, submissions: make(map[string][]GradientSubmission), audit: NewAuditLog()}
}

func (h *Hub) Submit(sub GradientSubmission) error {
	if len(sub.Vector) == 0 {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.submissions[sub.ZoneID] = append(h.submissions[sub.ZoneID], sub)
	if h.audit != nil {
		h.audit.Record(sub)
	}
	return nil
}

// AuditEntries возвращает копию журнала submissions.
func (h *Hub) AuditEntries() []SubmissionAudit {
	if h.audit == nil {
		return nil
	}
	return h.audit.Entries()
}

// Aggregate выполняет FedAvg + Laplace noise (differential privacy).
func (h *Hub) Aggregate() ([]float64, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var zones []string
	for z, subs := range h.submissions {
		if len(subs) > 0 {
			zones = append(zones, z)
		}
	}
	if len(zones) < 2 {
		return h.global, h.round
	}

	dim := len(h.submissions[zones[0]][len(h.submissions[zones[0]])-1].Vector)
	out := make([]float64, dim)
	weightSum := 0.0
	for _, z := range zones {
		sub := h.submissions[z][len(h.submissions[z])-1]
		w := float64(sub.SampleCount)
		if w <= 0 {
			w = 1
		}
		for i := 0; i < dim && i < len(sub.Vector); i++ {
			out[i] += sub.Vector[i] * w
		}
		weightSum += w
	}
	if weightSum > 0 {
		for i := range out {
			out[i] /= weightSum
			out[i] += laplaceNoise(1.0 / h.epsilon)
		}
	}
	h.global = out
	h.round++
	h.submissions = make(map[string][]GradientSubmission)
	return out, h.round
}

func (h *Hub) GlobalModel() ([]float64, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]float64, len(h.global))
	copy(out, h.global)
	return out, h.round
}

func laplaceNoise(scale float64) float64 {
	u := rand.Float64() - 0.5
	if u == 0 {
		u = 0.0001
	}
	sign := 1.0
	if u < 0 {
		sign = -1
	}
	return -scale * sign * math.Log(1-2*math.Abs(u))
}
