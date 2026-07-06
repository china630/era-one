// Package risk — CTEM risk score из VM + exposure (ADR-0006 G-03).
package risk

import (
	"strings"
)

var severityWeights = map[string]float64{
	"critical": 25,
	"high":     15,
	"medium":   8,
	"low":      3,
	"info":     1,
}

// Input — сигналы для расчёта risk score актива.
type Input struct {
	NodeID      string
	Platform    string
	CVECounts   map[string]int
	DetCounts   map[string]int
	Exposure    float64
	BASDetected bool
}

// Score — итоговый risk score с разбивкой.
type Score struct {
	NodeID         string  `json:"node_id"`
	Total          float64 `json:"total"`
	ExposureScore  float64 `json:"exposure_score"`
	BASFeedback    float64 `json:"bas_feedback"`
	VMContribution float64 `json:"vm_contribution"`
}

func weightedSum(counts map[string]int) float64 {
	var sum float64
	for sev, n := range counts {
		w := severityWeights[strings.ToLower(sev)]
		if w == 0 {
			w = 5
		}
		sum += w * float64(n)
	}
	return sum
}

func platformCriticality(platform string) float64 {
	p := strings.ToLower(platform)
	switch {
	case strings.Contains(p, "server"), strings.Contains(p, "dc"):
		return 1.5
	case strings.Contains(p, "windows"), strings.Contains(p, "linux"):
		return 1.2
	default:
		return 1.0
	}
}

// ComputeRisk считает risk = exposure + VM CVE вес + BAS feedback.
func ComputeRisk(in Input) Score {
	crit := platformCriticality(in.Platform)
	det := weightedSum(in.DetCounts)
	cve := weightedSum(in.CVECounts) * 1.2
	exposurePart := in.Exposure
	if exposurePart <= 0 {
		exposurePart = (det + cve) * crit
	}
	vmPart := cve
	basBonus := 0.0
	if in.BASDetected {
		basBonus = 20
	}
	final := exposurePart + vmPart + basBonus
	return Score{
		NodeID:         in.NodeID,
		Total:          final,
		ExposureScore:  exposurePart,
		BASFeedback:    basBonus,
		VMContribution: vmPart,
	}
}

// RankByRisk сортирует scores по убыванию total.
func RankByRisk(scores []Score) []Score {
	cp := append([]Score(nil), scores...)
	for i := 0; i < len(cp); i++ {
		for j := i + 1; j < len(cp); j++ {
			if cp[j].Total > cp[i].Total {
				cp[i], cp[j] = cp[j], cp[i]
			}
		}
	}
	return cp
}
