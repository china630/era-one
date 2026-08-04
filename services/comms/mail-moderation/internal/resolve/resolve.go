// Package resolve — выбор модератора (manager / attr / static / map).
package resolve

import (
	"fmt"
	"strings"

	"era/services/comms/mail-moderation/internal/policy"
)

// Directory — каталог отправителей.
type Directory interface {
	Manager(sender string) (string, error)
	Attr(sender, name string) (string, error)
	Curator(sender string) (string, error)
}

// MemoryDir — in-memory LDAP/PG stub.
type MemoryDir struct {
	Managers map[string]string
	Attrs    map[string]map[string]string // sender → attr → value
	Curators map[string]string
}

func (m *MemoryDir) Manager(sender string) (string, error) {
	if m == nil {
		return "", nil
	}
	return m.Managers[strings.ToLower(sender)], nil
}

func (m *MemoryDir) Attr(sender, name string) (string, error) {
	if m == nil {
		return "", nil
	}
	if a := m.Attrs[strings.ToLower(sender)]; a != nil {
		return a[name], nil
	}
	return "", nil
}

func (m *MemoryDir) Curator(sender string) (string, error) {
	if m == nil {
		return "", nil
	}
	return m.Curators[strings.ToLower(sender)], nil
}

// Resolver выбирает список модераторов (any-of).
type Resolver struct {
	Dir Directory
}

// Resolve возвращает 1+ адресов модераторов.
func (r *Resolver) Resolve(sender string, spec policy.ModeratorSpec) ([]string, error) {
	sender = strings.ToLower(strings.TrimSpace(sender))
	var out []string
	switch spec.Mode {
	case policy.ModStatic, policy.ModAllOf:
		// all_of: same address list; engine/quorum layer requires unanimous approve.
		out = append(out, spec.Static...)
	case policy.ModManager:
		m, err := r.Dir.Manager(sender)
		if err != nil {
			return nil, err
		}
		if m != "" {
			out = append(out, m)
		}
	case policy.ModLDAPAttr:
		if spec.LDAPAttr == "" {
			return nil, fmt.Errorf("ldap_attr empty")
		}
		v, err := r.Dir.Attr(sender, spec.LDAPAttr)
		if err != nil {
			return nil, err
		}
		if v != "" {
			out = append(out, v)
		}
	case policy.ModCuratorMap:
		v, err := r.Dir.Curator(sender)
		if err != nil {
			return nil, err
		}
		if v != "" {
			out = append(out, v)
		}
	default:
		return nil, fmt.Errorf("unknown moderator mode %q", spec.Mode)
	}
	if len(out) == 0 {
		out = append(out, spec.Fallback...)
	}
	cleaned := make([]string, 0, len(out))
	for _, a := range out {
		a = strings.TrimSpace(a)
		if a != "" {
			cleaned = append(cleaned, a)
		}
	}
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("no moderator for %s (mode=%s)", sender, spec.Mode)
	}
	return cleaned, nil
}
