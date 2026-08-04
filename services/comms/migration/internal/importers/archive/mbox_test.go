package archive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMBOXGolden(t *testing.T) {
	path := filepath.Join("testdata", "sample.mbox")
	msgs, err := ImportMBOX(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("want 2 messages got %d", len(msgs))
	}
	goldenPath := filepath.Join("testdata", "sample.mbox.golden.hash")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	var hashes strings.Builder
	for _, m := range msgs {
		hashes.WriteString(m.Hash)
		hashes.WriteByte('\n')
	}
	got := strings.ReplaceAll(hashes.String(), "\r", "")
	wantStr := strings.ReplaceAll(string(want), "\r", "")
	if strings.TrimSpace(got) != strings.TrimSpace(wantStr) {
		t.Fatalf("hash mismatch; got:\n%s", got)
	}
}
