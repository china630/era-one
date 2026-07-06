package adapters

import (
	"os"
	"testing"
)

func TestParsePRTGGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/prtg_webhook.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	w, err := ParsePRTG(raw)
	if err != nil {
		t.Fatal(err)
	}
	if w.NodeID() != "net-10-0-0-1" {
		t.Fatalf("node: %s", w.NodeID())
	}
	if !stringsContains(w.Summary(), "high egress") {
		t.Fatalf("summary: %s", w.Summary())
	}
}

func TestParseSyslogGolden(t *testing.T) {
	host, sum, err := ParseSyslogNetwork("10.0.0.5|PRTG high egress sensor alert")
	if err != nil {
		t.Fatal(err)
	}
	if host != "10.0.0.5" || !stringsContains(sum, "egress") {
		t.Fatalf("%s %s", host, sum)
	}
}

func TestFuzzSyslogNoPanic(t *testing.T) {
	for _, s := range []string{"", "|", "a|b|c", "\x00\x01", "host only"} {
		_, _, _ = ParseSyslogNetwork(s)
	}
}

func stringsContains(hay, needle string) bool {
	return len(needle) > 0 && (len(hay) >= len(needle)) && indexFold(hay, needle) >= 0
}

func indexFold(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
