package tip

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFeedFromSTIX(t *testing.T) {
	data := []byte(`{
		"type":"bundle","id":"bundle--1","spec_version":"2.1",
		"objects":[
			{"type":"indicator","id":"ind--1","pattern":"[domain-name:value = 'c2.evil.tld']","pattern_type":"stix"},
			{"type":"indicator","id":"ind--2","pattern":"deadbeefcafe","pattern_type":"stix"}
		]
	}`)
	feed, err := FeedFromSTIX(data)
	if err != nil {
		t.Fatal(err)
	}
	if feed.PatternCount() != 2 {
		t.Fatalf("patterns=%d", feed.PatternCount())
	}
	ok, rule := feed.Match(`{"dns":"c2.evil.tld"}`)
	if !ok || rule != "era-national-ioc" {
		t.Fatalf("match ok=%v rule=%s", ok, rule)
	}
}

func TestLoadSTIXBundleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	if err := os.WriteFile(path, []byte(`{
		"type":"bundle","objects":[{"type":"indicator","pattern":"[ipv4-addr:value = '10.0.0.66']"}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	feed, err := LoadSTIXBundle(path)
	if err != nil {
		t.Fatal(err)
	}
	if feed.PatternCount() != 1 {
		t.Fatalf("patterns=%d", feed.PatternCount())
	}
}
