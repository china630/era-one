package risk

import "testing"

func TestComputeRiskWithBASFeedback(t *testing.T) {
	base := ComputeRisk(Input{
		NodeID: "n1", Platform: "linux-server",
		CVECounts: map[string]int{"high": 2},
		DetCounts: map[string]int{"medium": 1},
	})
	withBAS := ComputeRisk(Input{
		NodeID: "n1", Platform: "linux-server",
		CVECounts: map[string]int{"high": 2},
		DetCounts: map[string]int{"medium": 1},
		BASDetected: true,
	})
	if withBAS.Total <= base.Total {
		t.Fatalf("BAS feedback: base=%v with=%v", base.Total, withBAS.Total)
	}
	if withBAS.BASFeedback != 20 {
		t.Fatalf("bas=%v", withBAS.BASFeedback)
	}
}

func TestRankByRisk(t *testing.T) {
	scores := []Score{
		{NodeID: "a", Total: 10},
		{NodeID: "b", Total: 50},
		{NodeID: "c", Total: 30},
	}
	ranked := RankByRisk(scores)
	if ranked[0].NodeID != "b" || ranked[2].NodeID != "a" {
		t.Fatalf("rank: %+v", ranked)
	}
}
