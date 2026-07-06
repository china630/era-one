// Golden MITRE eval scenarios (S6-18).
package mitreval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"era/services/detection-engine/internal/itdr"
	"era/services/detection-engine/internal/ndr"
)

type scenarioFile struct {
	Scenarios []scenario `json:"scenarios"`
}

type scenario struct {
	ID           string           `json:"id"`
	Technique    string           `json:"technique"`
	ExpectedRule string           `json:"expected_rule"`
	Engine       string           `json:"engine"`
	Events       []map[string]any `json:"events"`
	BeaconCount  int              `json:"beacon_count"`
	BeaconSec    int              `json:"beacon_interval_sec"`
	DNSQueries   int              `json:"dns_queries"`
	ExfilChunks  int              `json:"exfil_chunks"`
	ExfilBytes   int              `json:"exfil_chunk_bytes"`
}

func TestMITREEvalScenariosGolden(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "data", "mitre-eval", "scenarios.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc scenarioFile
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Scenarios) < 6 {
		t.Fatalf("scenarios=%d", len(doc.Scenarios))
	}
	for _, sc := range doc.Scenarios {
		switch sc.Engine {
		case "itdr":
			runITDR(t, sc)
		case "ndr":
			runNDR(t, sc)
		default:
			t.Fatalf("unknown engine %s", sc.Engine)
		}
	}
}

func runITDR(t *testing.T, sc scenario) {
	for _, ev := range sc.Events {
		body, _ := json.Marshal(ev)
		ok, rule := itdr.MatchAuth(string(body))
		if !ok || rule.ID != sc.ExpectedRule {
			t.Fatalf("%s: got ok=%v rule=%s want %s", sc.ID, ok, rule.ID, sc.ExpectedRule)
		}
	}
}

func runNDR(t *testing.T, sc scenario) {
	e := ndr.New(30 * time.Minute)
	at := time.Now().UTC()
	switch sc.ID {
	case "eval-t1021-lateral":
		for _, ev := range sc.Events {
			src, _ := ev["src_ip"].(string)
			dst, _ := ev["dst_ip"].(string)
			port := uint32(0)
			if p, ok := ev["dst_port"].(float64); ok {
				port = uint32(p)
			}
			e.Observe(src, dst, port, at)
		}
		src, _ := sc.Events[0]["src_ip"].(string)
		ok, rule := e.LateralMovement(src)
		if !ok || rule != sc.ExpectedRule {
			t.Fatalf("%s lateral: ok=%v rule=%s", sc.ID, ok, rule)
		}
	case "eval-t1071-beacon":
		src, dst := "10.1.1.5", "203.0.113.9"
		n := sc.BeaconCount
		if n == 0 {
			n = 6
		}
		interval := sc.BeaconSec
		if interval == 0 {
			interval = 60
		}
		for i := 0; i < n; i++ {
			e.ObserveBeacon(src, dst, at.Add(time.Duration(i*interval)*time.Second))
		}
		ok, rule := e.Beaconing(src, dst)
		if !ok || rule != sc.ExpectedRule {
			t.Fatalf("%s beacon: ok=%v rule=%s", sc.ID, ok, rule)
		}
	case "eval-t1071-dns-tunnel":
		src := "10.1.1.8"
		n := sc.DNSQueries
		if n == 0 {
			n = 10
		}
		for i := 0; i < n; i++ {
			q := "abcdefghijklmnopqrstuvwxyz0123456789abcdef.example.com"
			e.ObserveDNS(src, q, at.Add(time.Duration(i)*time.Second))
		}
		ok, rule := e.DNSTunnel(src)
		if !ok || rule != sc.ExpectedRule {
			t.Fatalf("%s dns: ok=%v rule=%s", sc.ID, ok, rule)
		}
	case "eval-t1048-exfil":
		src := "10.2.0.5"
		chunks := sc.ExfilChunks
		if chunks == 0 {
			chunks = 5
		}
		chunk := uint64(sc.ExfilBytes)
		if chunk == 0 {
			chunk = 12 * 1024 * 1024
		}
		for i := 0; i < chunks; i++ {
			e.ObserveExfil(src, chunk, "", at.Add(time.Duration(i)*time.Second))
		}
		ok, rule := e.DataExfil(src)
		if !ok || rule != sc.ExpectedRule {
			t.Fatalf("%s exfil: ok=%v rule=%s", sc.ID, ok, rule)
		}
	default:
		t.Fatalf("unsupported ndr scenario %s", sc.ID)
	}
}
