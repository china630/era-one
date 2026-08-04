package investigate

import (
	"encoding/json"

	"era/services/platform/custody"
)

// SealEvidence seals storyline event IDs into a custody chain (ADR-0023 AI-5).
func SealEvidence(chain *custody.Chain, res *Result) string {
	if chain == nil || res == nil {
		return ""
	}
	for _, step := range res.Storyline {
		payload, _ := json.Marshal(map[string]string{
			"event_id": step.EventID,
			"category": step.Category,
			"verdict":  res.Verdict,
		})
		_ = chain.Seal(payload)
	}
	summary, _ := json.Marshal(map[string]any{
		"detection_id": res.DetectionID,
		"verdict":      res.Verdict,
		"prompt_hash":  res.PromptHash,
		"events":       len(res.Storyline),
	})
	entry := chain.Seal(summary)
	res.CustodyRootHash = entry.Hash
	return entry.Hash
}
