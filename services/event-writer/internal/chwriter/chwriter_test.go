package chwriter

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMapInventoryEnvelopeGolden(t *testing.T) {
	env := testInventoryEnvelope()
	row, err := mapInventoryEnvelope(env)
	if err != nil {
		t.Fatal(err)
	}
	got, err := inventoryRowJSON(row)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join("testdata", "inventory_row.golden.json")
	want, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("inventory row mismatch; got:\n%s\nwant:\n%s", got, want)
	}
}

func TestIsInventoryEnvelope(t *testing.T) {
	env := testInventoryEnvelope()
	if !isInventoryEnvelope(env) {
		t.Fatal("expected inventory envelope")
	}
}
