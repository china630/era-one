package foldermap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"era/services/comms/internal/imapclient"
)

func TestCGFolderTreeGolden(t *testing.T) {
	got := CGFixtureGolden()
	want := loadGoldenTree(t)
	for src, target := range want {
		if got[src] != target {
			t.Fatalf("Resolve(%q)=%q want %q", src, got[src], target)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("tree size got %d want %d", len(got), len(want))
	}
}

func loadGoldenTree(t *testing.T) map[string]string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "testdata", "cg_iw_tree.golden.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSpecialUseOverridesPath(t *testing.T) {
	mb := imapclientMailbox(`\Sent`, "Sent Items")
	if got := Resolve(mb, nil); got != "Sent" {
		t.Fatalf("got %q want Sent", got)
	}
}

func imapclientMailbox(attrs ...string) imapclient.Mailbox {
	name := attrs[len(attrs)-1]
	flags := attrs[:len(attrs)-1]
	return imapclient.Mailbox{Name: name, Attributes: flags}
}
