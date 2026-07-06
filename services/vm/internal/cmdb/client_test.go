package cmdb

import (
	"testing"
)

func TestMatchProducts(t *testing.T) {
	rows := []SoftwareRow{
		{NodeID: "n1", Name: "openssl", Version: "3.0"},
		{NodeID: "n2", Name: "curl", Version: "8.5"},
	}
	m := MatchProducts(rows, "ssl")
	if len(m) != 1 || m[0].Name != "openssl" {
		t.Fatalf("matches=%v", m)
	}
}
