package cvefeed

import (
	"path/filepath"
	"testing"

	"era/services/vm/internal/cmdb"
)

func TestMatchSoftwareGolden(t *testing.T) {
	feed, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "cve-feed", "cve.json"))
	if err != nil {
		t.Fatal(err)
	}
	rows := []cmdb.SoftwareRow{
		{NodeID: "n1", Name: "OpenSSL", Version: "3.0.10"},
		{NodeID: "n1", Name: "OpenSSL", Version: "3.0.13"},
		{NodeID: "n2", Name: "curl", Version: "8.0.0"},
	}
	findings := MatchSoftware(feed, rows)
	if len(findings) != 1 {
		t.Fatalf("findings=%d want 1 (only 3.0.10 < 3.0.13)", len(findings))
	}
	if findings[0].TemplateID != "CVE-2024-0001" || findings[0].Target != "n1" {
		t.Fatalf("%+v", findings[0])
	}
}

func TestVersionLess(t *testing.T) {
	if !versionLess("3.0.10", "3.0.13") {
		t.Fatal("expected less")
	}
	if versionLess("3.0.13", "3.0.13") {
		t.Fatal("equal not less")
	}
	if versionLess("3.1.0", "3.0.13") {
		t.Fatal("greater")
	}
}
