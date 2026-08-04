package mitre

import (
	"era/services/detection-engine/internal/sigma"
)

// TechniqueCoverage is one ATT&CK technique row for the heatmap.
type TechniqueCoverage struct {
	Technique string   `json:"technique"`
	RuleIDs   []string `json:"rule_ids"`
	RuleCount int      `json:"rule_count"`
	InCorpus  bool     `json:"in_corpus"`
	SeenCount int      `json:"seen_count"`
}

// CorpusCoverage aggregates Sigma tags into technique coverage.
func CorpusCoverage(rules []*sigma.Rule) []TechniqueCoverage {
	by := map[string][]string{}
	for _, r := range rules {
		if r == nil {
			continue
		}
		for _, t := range r.Techniques() {
			by[t] = append(by[t], r.ID)
		}
	}
	out := make([]TechniqueCoverage, 0, len(by))
	for tech, ids := range by {
		out = append(out, TechniqueCoverage{
			Technique: tech,
			RuleIDs:   ids,
			RuleCount: len(ids),
			InCorpus:  true,
		})
	}
	// stable-ish order by technique id
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Technique < out[i].Technique {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// MergeSeen overlays runtime hit counts onto corpus coverage.
func MergeSeen(cov []TechniqueCoverage, seen map[string]int) []TechniqueCoverage {
	idx := map[string]int{}
	for i, c := range cov {
		idx[c.Technique] = i
	}
	for tech, n := range seen {
		if i, ok := idx[tech]; ok {
			cov[i].SeenCount = n
			continue
		}
		cov = append(cov, TechniqueCoverage{Technique: tech, SeenCount: n, InCorpus: false})
	}
	return cov
}
