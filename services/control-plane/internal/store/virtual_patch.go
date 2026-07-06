package store

import (
	"fmt"
	"strings"
)

// VirtualPatchFromFinding — правило virtual patch из CVE/hash finding (ADR-0017 §4).
func VirtualPatchFromFinding(cveID, hashSHA256, path, vector string) (EnforcementPolicy, error) {
	cveID = strings.TrimSpace(cveID)
	hashSHA256 = strings.TrimSpace(strings.ToLower(hashSHA256))
	path = strings.TrimSpace(path)
	vector = strings.TrimSpace(vector)
	if cveID == "" && hashSHA256 == "" {
		return EnforcementPolicy{}, fmt.Errorf("cve_id or hash_sha256 required")
	}
	if cveID == "" {
		if len(hashSHA256) < 12 {
			return EnforcementPolicy{}, fmt.Errorf("hash_sha256 too short")
		}
		cveID = "hash-" + hashSHA256[:12]
	}
	ruleID := "vp-" + strings.ReplaceAll(strings.ToLower(cveID), ":", "-")
	vp := VirtualPatchRule{
		ID: ruleID, CVEID: cveID, Action: "deny", Path: path, Vector: vector,
		HashSHA256: hashSHA256,
	}
	return EnforcementPolicy{
		VirtualPatches: []VirtualPatchRule{vp},
	}, nil
}

// MergeVirtualPatch добавляет virtual patch + hash app_rule в существующую policy.
func MergeVirtualPatch(cur EnforcementPolicy, cveID, hashSHA256, path, vector string) (EnforcementPolicy, error) {
	patch, err := VirtualPatchFromFinding(cveID, hashSHA256, path, vector)
	if err != nil {
		return cur, err
	}
	vp := patch.VirtualPatches[0]
	out := cur
	out.VirtualPatches = append(append([]VirtualPatchRule{}, cur.VirtualPatches...), vp)
	if hashSHA256 != "" {
		out.AppRules = append(append([]EnforcementAppRule{}, cur.AppRules...), EnforcementAppRule{
			ID: "vp-hash-" + vp.ID, Action: "deny", HashSHA256: hashSHA256,
		})
	}
	return out, nil
}
