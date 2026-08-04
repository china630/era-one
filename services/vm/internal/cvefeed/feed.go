// Package cvefeed — offline CVE feed matcher for VM (ADR-0022 DC-04).
package cvefeed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"era/services/vm/internal/cmdb"
	"era/services/vm/internal/models"
)

// Entry is one CVE in the feed.
type Entry struct {
	ID        string `json:"id"`
	Product   string `json:"product"`
	VersionLt string `json:"version_lt"`
	Severity  string `json:"severity"`
	Summary   string `json:"summary"`
}

// Feed is an offline CVE pack.
type Feed struct {
	Version string  `json:"version"`
	CVEs    []Entry `json:"cves"`
}

// LoadFile loads a single JSON feed.
func LoadFile(path string) (*Feed, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f Feed
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// LoadDir loads all *.json feeds from a directory.
func LoadDir(dir string) (*Feed, error) {
	merged := &Feed{Version: "merged"}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		f, err := LoadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		merged.CVEs = append(merged.CVEs, f.CVEs...)
		if f.Version != "" {
			merged.Version = f.Version
		}
	}
	return merged, nil
}

// MatchSoftware returns findings for installed software older than version_lt.
func MatchSoftware(f *Feed, rows []cmdb.SoftwareRow) []models.Finding {
	if f == nil {
		return nil
	}
	var out []models.Finding
	now := time.Now().UTC()
	for _, cve := range f.CVEs {
		for _, sw := range cmdb.MatchProducts(rows, cve.Product) {
			if versionLess(sw.Version, cve.VersionLt) {
				out = append(out, models.Finding{
					TemplateID:        cve.ID,
					Target:            sw.NodeID,
					Severity:          cve.Severity,
					VulnerabilityName: cve.ID + " " + sw.Name + " " + sw.Version,
					MatchedURL:        cve.Summary,
					Timestamp:         now,
				})
			}
		}
	}
	return out
}

// versionLess is a simple dotted numeric compare (MVP).
func versionLess(cur, limit string) bool {
	if limit == "" {
		return true
	}
	a := splitVer(cur)
	b := splitVer(limit)
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(a) {
			ai = a[i]
		}
		if i < len(b) {
			bi = b[i]
		}
		if ai < bi {
			return true
		}
		if ai > bi {
			return false
		}
	}
	return false
}

func splitVer(s string) []int {
	s = strings.TrimSpace(s)
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			break
		}
		out = append(out, n)
	}
	return out
}
