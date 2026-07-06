// Package exposure — per-asset risk score (ADR-0017 §2): CVE + детекты + критичность.
package exposure

import (
	"sort"
	"strings"
)

// SeverityWeights — те же веса, что risk.Scorer (S6-7).
var SeverityWeights = map[string]float64{
	"critical": 25,
	"high":     15,
	"medium":   8,
	"low":      3,
	"info":     1,
}

// AssetExposure — итоговый exposure score на актив.
type AssetExposure struct {
	NodeID          string  `json:"node_id"`
	Hostname        string  `json:"hostname,omitempty"`
	Platform        string  `json:"platform,omitempty"`
	Score           float64 `json:"score"`
	DetectionScore  float64 `json:"detection_score"`
	CVEScore        float64 `json:"cve_score"`
	Criticality     float64 `json:"criticality"`
	MisconfigScore  float64 `json:"misconfig_score,omitempty"`
	DetectionCounts map[string]int `json:"detection_counts,omitempty"`
	CVECounts       map[string]int `json:"cve_counts,omitempty"`
}

func weightedSum(counts map[string]int) float64 {
	var sum float64
	for sev, n := range counts {
		w := SeverityWeights[strings.ToLower(sev)]
		if w == 0 {
			w = 5
		}
		sum += w * float64(n)
	}
	return sum
}

// PlatformCriticality — базовая критичность до полного CMDB (Этап 5).
func PlatformCriticality(platform string) float64 {
	p := strings.ToLower(platform)
	switch {
	case strings.Contains(p, "server"), strings.Contains(p, "dc"), strings.Contains(p, "domain"):
		return 1.5
	case strings.Contains(p, "windows"), strings.Contains(p, "linux"):
		return 1.2
	default:
		return 1.0
	}
}

// ComputeScore считает exposure = (detections + cve*1.2 + misconfig) * criticality.
func ComputeScore(detCounts, cveCounts map[string]int, criticality, misconfig float64) (total, det, cve float64) {
	det = weightedSum(detCounts)
	cve = weightedSum(cveCounts) * 1.2
	if criticality <= 0 {
		criticality = 1.0
	}
	total = (det + cve + misconfig) * criticality
	return
}

// BuildAssets объединяет сигналы по node_id.
func BuildAssets(
	nodes map[string]map[string]int,
	cves map[string]map[string]int,
	meta map[string]AssetMeta,
) []AssetExposure {
	seen := make(map[string]struct{})
	for n := range nodes {
		seen[n] = struct{}{}
	}
	for n := range cves {
		seen[n] = struct{}{}
	}
	for n := range meta {
		seen[n] = struct{}{}
	}
	out := make([]AssetExposure, 0, len(seen))
	for nodeID := range seen {
		det := cloneCounts(nodes[nodeID])
		cve := cloneCounts(cves[nodeID])
		m := meta[nodeID]
		crit := m.Criticality
		if crit <= 0 {
			crit = PlatformCriticality(m.Platform)
		}
		total, detScore, cveScore := ComputeScore(det, cve, crit, m.Misconfig)
		out = append(out, AssetExposure{
			NodeID:          nodeID,
			Hostname:        m.Hostname,
			Platform:        m.Platform,
			Score:           total,
			DetectionScore:  detScore,
			CVEScore:        cveScore,
			Criticality:     crit,
			MisconfigScore:  m.Misconfig,
			DetectionCounts: det,
			CVECounts:       cve,
		})
	}
	return out
}

// AssetMeta — метаданные актива из control-plane.
type AssetMeta struct {
	Hostname    string
	Platform    string
	Criticality float64
	Misconfig   float64
}

// TopN возвращает топ-N активов по exposure score (desc).
func TopN(assets []AssetExposure, n int) []AssetExposure {
	cp := append([]AssetExposure(nil), assets...)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].Score == cp[j].Score {
			return cp[i].NodeID < cp[j].NodeID
		}
		return cp[i].Score > cp[j].Score
	})
	if n <= 0 || n > len(cp) {
		n = len(cp)
	}
	if n == 0 {
		return nil
	}
	return cp[:n]
}

func cloneCounts(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
