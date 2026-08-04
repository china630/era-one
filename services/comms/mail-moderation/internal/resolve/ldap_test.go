package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"era/services/comms/mail-moderation/internal/resolve"
)

func TestLoadLDAPFromJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ldap.json")
	content := `{"managers":{"alex@company.local":"ivan@company.local"},"groups":{"alex@company.local":["novices"]},"attrs":{"alex@company.local":{"extensionAttribute1":"sergey@company.local"}}}`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := resolve.LoadLDAPFromJSON(p)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := d.Manager("alex@company.local")
	if m != "ivan@company.local" {
		t.Fatal(m)
	}
	if g := d.Groups("alex@company.local"); len(g) != 1 || g[0] != "novices" {
		t.Fatalf("%v", g)
	}
}
