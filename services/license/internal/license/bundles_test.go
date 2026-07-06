package license

import "testing"

func TestModulesForBundleCoreAIResponse(t *testing.T) {
	mods, err := ModulesForBundle(BundleCoreAIResponse)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 2 || mods[0] != ModuleControlAI || mods[1] != ModuleResponse {
		t.Fatalf("unexpected: %v", mods)
	}
}

func TestModulesForBundleGA1Alias(t *testing.T) {
	mods, err := ModulesForBundle("ga1")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 2 {
		t.Fatal(mods)
	}
}
