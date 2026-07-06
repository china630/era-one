package compliance

import "testing"

func TestComplianceReportGolden(t *testing.T) {
	lines := BuildReport("bank-az", "2026-Q2", 1200, 5)
	if len(lines) != 3 || lines[2].Value != 1205 {
		t.Fatalf("lines=%+v", lines)
	}
	if Summary("bank-az", "2026-Q2", 1200, 5) != "bank-az/2026-Q2 events=1200 cases=5" {
		t.Fatal("summary mismatch")
	}
}
