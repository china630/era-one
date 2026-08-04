package licensegate

import "testing"

func TestOfficeTables(t *testing.T) {
	g := FromModules([]Module{ModuleOfficeTables})
	if !g.Allow(ModuleOfficeTables) {
		t.Fatal("expected office-tables allowed")
	}
	g2 := FromModules(nil)
	if g2.Allow(ModuleOfficeTables) {
		t.Fatal("expected deny without module")
	}
}

func TestOfficePresentations(t *testing.T) {
	g := FromModules([]Module{ModuleOfficePresentations})
	if !g.Allow(ModuleOfficePresentations) {
		t.Fatal("expected office-presentations")
	}
}

func TestOfficeProjects(t *testing.T) {
	if !OfficeDevGate().Allow(ModuleOfficeProjects) {
		t.Fatal("expected office-projects in OfficeDevGate")
	}
}

func TestOfficeAI(t *testing.T) {
	if !OfficeDevGate().Allow(ModuleOfficeAI) {
		t.Fatal("expected office-ai in OfficeDevGate")
	}
}
