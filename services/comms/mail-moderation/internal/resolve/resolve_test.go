package resolve_test

import (
	"testing"

	"era/services/comms/mail-moderation/internal/policy"
	"era/services/comms/mail-moderation/internal/resolve"
)

func TestResolve_ManagerAndFallback(t *testing.T) {
	dir := &resolve.MemoryDir{
		Managers: map[string]string{"alex@company.local": "ivan@company.local"},
	}
	r := &resolve.Resolver{Dir: dir}
	mods, err := r.Resolve("alex@company.local", policy.ModeratorSpec{
		Mode:     policy.ModManager,
		Fallback: []string{"hr@company.local"},
	})
	if err != nil || mods[0] != "ivan@company.local" {
		t.Fatalf("got %v %v", mods, err)
	}
	mods, err = r.Resolve("nobody@company.local", policy.ModeratorSpec{
		Mode:     policy.ModManager,
		Fallback: []string{"hr@company.local"},
	})
	if err != nil || mods[0] != "hr@company.local" {
		t.Fatalf("fallback got %v %v", mods, err)
	}
}

func TestResolve_AttrOverridesManagerSemantics(t *testing.T) {
	dir := &resolve.MemoryDir{
		Managers: map[string]string{"alex@company.local": "ivan@company.local"},
		Attrs: map[string]map[string]string{
			"alex@company.local": {"extensionAttribute1": "sergey@company.local"},
		},
	}
	r := &resolve.Resolver{Dir: dir}
	mods, err := r.Resolve("alex@company.local", policy.ModeratorSpec{
		Mode:     policy.ModLDAPAttr,
		LDAPAttr: "extensionAttribute1",
	})
	if err != nil || mods[0] != "sergey@company.local" {
		t.Fatalf("want sergey, got %v %v", mods, err)
	}
}

func TestResolve_CuratorMap(t *testing.T) {
	dir := &resolve.MemoryDir{
		Curators: map[string]string{"alex@company.local": "curator@other.local"},
	}
	r := &resolve.Resolver{Dir: dir}
	mods, err := r.Resolve("alex@company.local", policy.ModeratorSpec{Mode: policy.ModCuratorMap})
	if err != nil || mods[0] != "curator@other.local" {
		t.Fatalf("got %v %v", mods, err)
	}
}

func TestResolve_StaticAnyOf(t *testing.T) {
	r := &resolve.Resolver{Dir: &resolve.MemoryDir{}}
	mods, err := r.Resolve("x@y.z", policy.ModeratorSpec{
		Mode:   policy.ModStatic,
		Static: []string{"a@c.local", "b@c.local"},
	})
	if err != nil || len(mods) != 2 {
		t.Fatalf("got %v %v", mods, err)
	}
}
