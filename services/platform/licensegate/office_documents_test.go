package licensegate

import "testing"

func TestOfficeDocuments(t *testing.T) {
	g := FromModules([]Module{ModuleOfficeDocuments})
	if !g.Allow(ModuleOfficeDocuments) {
		t.Fatal("office-documents should be enabled")
	}
	g2 := FromModules(nil)
	if g2.Allow(ModuleOfficeDocuments) {
		t.Fatal("office-documents should be disabled without module")
	}
}

func TestOfficeMvpBundle(t *testing.T) {
	g := FromModules([]Module{ModulePlatformDrive, ModuleOfficeDocuments})
	if !g.Allow(ModulePlatformDrive) || !g.Allow(ModuleOfficeDocuments) {
		t.Fatal("office-mvp bundle needs platform-drive and office-documents")
	}
}
