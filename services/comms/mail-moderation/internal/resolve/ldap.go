package resolve

import (
	"encoding/json"
	"os"
	"strings"
)

// LDAPDir — directory adapter. Без живого LDAP в CI: загружает JSON из ERA_MM_LDAP_JSON
// или использует MemoryDir. Поля совместимы с manager/attr/group lookups.
type LDAPDir struct {
	*MemoryDir
	GroupMap map[string][]string // sender → groups
}

// Groups реализует engine.GroupLookup.
func (l *LDAPDir) Groups(sender string) []string {
	if l == nil {
		return nil
	}
	s := strings.ToLower(strings.TrimSpace(sender))
	if l.GroupMap != nil {
		return append([]string(nil), l.GroupMap[s]...)
	}
	return nil
}

// LDAPSnapshot — JSON shape для air-gap/lab dump.
type LDAPSnapshot struct {
	Managers map[string]string            `json:"managers"`
	Attrs    map[string]map[string]string `json:"attrs"`
	Curators map[string]string            `json:"curators"`
	Groups   map[string][]string          `json:"groups"`
}

func LoadLDAPFromJSON(path string) (*LDAPDir, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap LDAPSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, err
	}
	return &LDAPDir{
		MemoryDir: &MemoryDir{Managers: snap.Managers, Attrs: snap.Attrs, Curators: snap.Curators},
		GroupMap:  lowerGroups(snap.Groups),
	}, nil
}

func lowerGroups(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[strings.ToLower(k)] = v
	}
	return out
}

// DirectoryFromEnv — LDAP JSON если ERA_MM_LDAP_JSON, иначе MemoryDir.
func DirectoryFromEnv() Directory {
	if p := os.Getenv("ERA_MM_LDAP_JSON"); p != "" {
		if d, err := LoadLDAPFromJSON(p); err == nil {
			return d
		}
	}
	return &MemoryDir{
		Managers: map[string]string{},
		Attrs:    map[string]map[string]string{},
		Curators: map[string]string{},
	}
}

// GroupLookupFromDir — если LDAPDir, иначе empty.
func GroupLookupFromDir(d Directory) interface{ Groups(string) []string } {
	if l, ok := d.(*LDAPDir); ok {
		return l
	}
	if o, ok := d.(*OverlayDir); ok {
		if l, ok := o.Base.(*LDAPDir); ok {
			return l
		}
	}
	return emptyGroups{}
}

type emptyGroups struct{}

func (emptyGroups) Groups(string) []string { return nil }
