package guard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"era/services/resolve/internal/atlas"
	"era/services/resolve/internal/policy"
)

func TestDecidePolicyAndAtlas(t *testing.T) {
	pol := policy.NewStore()
	atl := atlas.New()
	packPath := filepath.Join("..", "atlas", "testdata", "atlas_pack.golden.json")
	if err := atl.LoadFile(packPath); err != nil {
		t.Fatal(err)
	}
	e := New(pol, atl)

	v := e.Decide("foo.malware.test", "A")
	if v.Action != policy.ActionNXDomain || v.Source != "policy" {
		t.Fatalf("%+v", v)
	}
	v = e.Decide("bar.phish.test", "A")
	if v.Action != policy.ActionSinkhole || v.Sinkhole == "" {
		t.Fatalf("%+v", v)
	}
	v = e.Decide("bad.actor.example", "A")
	if v.Action != policy.ActionNXDomain || v.Source != "atlas" {
		t.Fatalf("%+v", v)
	}
	v = e.Decide("www.example.com", "A")
	if v.Action != policy.ActionAllow {
		t.Fatalf("%+v", v)
	}

	got, _ := json.Marshal(map[string]any{
		"malware": e.Decide("x.malware.test", "A"),
		"atlas":   e.Decide("c2.evil.example", "A"),
		"allow":   e.Decide("ok.example", "A"),
	})
	golden := filepath.Join("testdata", "verdicts.golden.json")
	want, err := os.ReadFile(golden)
	if err != nil {
		_ = os.MkdirAll("testdata", 0o755)
		_ = os.WriteFile(golden, got, 0o644)
		t.Logf("wrote golden %s", golden)
		return
	}
	var a, b any
	_ = json.Unmarshal(got, &a)
	_ = json.Unmarshal(want, &b)
	ag, _ := json.Marshal(a)
	bg, _ := json.Marshal(b)
	if string(ag) != string(bg) {
		t.Fatalf("golden mismatch\ngot  %s\nwant %s", ag, bg)
	}
}
