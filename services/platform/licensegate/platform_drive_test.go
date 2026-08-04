package licensegate

import "testing"

func TestPlatformDrive(t *testing.T) {
	g := FromModules([]Module{ModulePlatformDrive})
	if !g.Allow(ModulePlatformDrive) {
		t.Fatal("platform-drive should be enabled")
	}
	g2 := FromModules(nil)
	if g2.Allow(ModulePlatformDrive) {
		t.Fatal("platform-drive should be disabled without module")
	}
}

func TestOfficeDevGateIncludesPlatformDrive(t *testing.T) {
	g := OfficeDevGate()
	if !g.Allow(ModulePlatformDrive) {
		t.Fatal("office dev gate needs platform-drive")
	}
}
