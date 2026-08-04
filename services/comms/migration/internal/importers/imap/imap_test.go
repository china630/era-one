package imap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportMailbox(t *testing.T) {
	file := filepath.Join("testdata", "messages.golden.txt")
	items, err := ImportMailbox(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}
	if os.Getenv("ERA_UPDATE_GOLDEN") == "1" {
		_ = os.WriteFile(file, []byte("msg-001\nmsg-002\nmsg-003\n"), 0o644)
	}
}
