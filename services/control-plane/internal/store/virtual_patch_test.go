package store

import "testing"

func TestMergeVirtualPatchFromCVE(t *testing.T) {
	cur := DefaultEnforcementPolicy()
	merged, err := MergeVirtualPatch(cur, "CVE-2024-9999", "", `*\vuln.dll`, "dll_load")
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.VirtualPatches) != len(cur.VirtualPatches)+1 {
		t.Fatalf("virtual patches: got %d want %d", len(merged.VirtualPatches), len(cur.VirtualPatches)+1)
	}
	last := merged.VirtualPatches[len(merged.VirtualPatches)-1]
	if last.CVEID != "CVE-2024-9999" {
		t.Fatalf("cve: %s", last.CVEID)
	}
}

func TestMergeVirtualPatchFromHash(t *testing.T) {
	cur := DefaultEnforcementPolicy()
	hash := "abcd1234ef567890abcd1234ef567890abcd1234ef567890abcd1234ef56"
	merged, err := MergeVirtualPatch(cur, "", hash, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.AppRules) != len(cur.AppRules)+1 {
		t.Fatal("expected hash app rule")
	}
	if merged.AppRules[len(merged.AppRules)-1].HashSHA256 != hash {
		t.Fatal("hash rule mismatch")
	}
}
