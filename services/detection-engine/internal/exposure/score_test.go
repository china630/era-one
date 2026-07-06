package exposure

import "testing"

func TestComputeScoreAndRank(t *testing.T) {
	detHigh := map[string]int{"high": 2}
	cveCrit := map[string]int{"critical": 1}
	total, det, cve := ComputeScore(detHigh, cveCrit, 1.5, 0)
	wantDet := 30.0
	wantCve := 30.0
	if det != wantDet || cve != wantCve {
		t.Fatalf("det=%v cve=%v want %v %v", det, cve, wantDet, wantCve)
	}
	wantTotal := (wantDet + wantCve) * 1.5
	if total != wantTotal {
		t.Fatalf("total=%v want %v", total, wantTotal)
	}

	assets := BuildAssets(
		map[string]map[string]int{
			"n-low":  {"low": 1},
			"n-high": {"critical": 1, "high": 1},
		},
		map[string]map[string]int{
			"n-high": {"medium": 2},
		},
		map[string]AssetMeta{
			"n-low":  {Platform: "linux"},
			"n-high": {Platform: "windows-server", Hostname: "dc01"},
		},
	)
	top := TopN(assets, 10)
	if len(top) != 2 {
		t.Fatalf("assets=%d", len(top))
	}
	if top[0].NodeID != "n-high" {
		t.Fatalf("expected n-high first, got %s score=%v", top[0].NodeID, top[0].Score)
	}
	if top[0].Hostname != "dc01" {
		t.Fatalf("hostname=%q", top[0].Hostname)
	}
}

func TestPlatformCriticality(t *testing.T) {
	if PlatformCriticality("windows-server") <= PlatformCriticality("unknown") {
		t.Fatal("server should be more critical")
	}
}
