package risk

import "time"

// ShouldEscalateCase — G-05: высокий risk score → эскалация в case.
func ShouldEscalateCase(score float64, threshold float64) bool {
	if threshold <= 0 {
		threshold = 30
	}
	return score >= threshold
}

// EscalationDecision для golden-тестов.
type EscalationDecision struct {
	NodeID    string  `json:"node_id"`
	Score     float64 `json:"score"`
	Escalate  bool    `json:"escalate"`
	Threshold float64 `json:"threshold"`
}

func DecideEscalation(nodeID string, scorer *Scorer, at time.Time, severities []string) EscalationDecision {
	for _, sev := range severities {
		scorer.Bump(nodeID, sev, at)
	}
	score := scorer.Score(nodeID)
	th := 30.0
	return EscalationDecision{
		NodeID: nodeID, Score: score,
		Escalate: ShouldEscalateCase(score, th), Threshold: th,
	}
}
