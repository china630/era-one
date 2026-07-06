package netflow

import (
	"os"
	"testing"
)

func TestParseLineGolden(t *testing.T) {
	raw, _ := os.ReadFile("testdata/flow_line.golden.txt")
	r, err := ParseLine(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if r.SrcIP != "10.0.0.10" || r.DstPort != 443 {
		t.Fatalf("%+v", r)
	}
}
