// Edition matrix — automated table-driven tests (S7-15, ADR-0005/0010).
package licensegate

import "testing"

func TestEditionMatrix(t *testing.T) {
	tests := []struct {
		name     string
		modules  []Module
		enabled  []Module
		disabled []Module
	}{
		{
			name:     "core_only",
			modules:  []Module{},
			enabled:  []Module{},
			disabled: KnownModules,
		},
		{
			name:     "core_plus_vuln_ai_response",
			modules:  []Module{ModuleVuln, ModuleControlAI, ModuleResponse},
			enabled:  []Module{ModuleVuln, ModuleControlAI, ModuleResponse},
			disabled: []Module{ModuleFederated, ModuleNational},
		},
		{
			name:     "era_national_edition",
			modules:  []Module{ModuleVuln, ModuleControlAI, ModuleResponse, ModuleNational},
			enabled:  []Module{ModuleNational},
			disabled: []Module{ModuleFederated},
		},
		{
			name:     "full_catalog",
			modules:  KnownModules,
			enabled:  KnownModules,
			disabled: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := FromModules(tc.modules)
			for _, m := range tc.enabled {
				if !g.Allow(m) {
					t.Fatalf("expected %s enabled", m)
				}
			}
			for _, m := range tc.disabled {
				if g.Allow(m) {
					t.Fatalf("expected %s disabled", m)
				}
			}
		})
	}
}

func TestDevDefaultMatchesCoreUpsell(t *testing.T) {
	g := DevDefault()
	for _, m := range []Module{ModuleVuln, ModuleControlAI, ModuleResponse, ModuleManage, ModuleService, ModuleProvision, ModulePAM, ModuleObserve} {
		if !g.Allow(m) {
			t.Fatalf("dev default missing %s", m)
		}
	}
	for _, m := range []Module{ModuleFederated, ModuleNational} {
		if g.Allow(m) {
			t.Fatalf("dev default must not include %s", m)
		}
	}
}

func TestBundleITOps(t *testing.T) {
	g := FromModules([]Module{ModuleManage, ModuleService, ModuleProvision})
	for _, m := range []Module{ModuleManage, ModuleService, ModuleProvision} {
		if !g.Allow(m) {
			t.Fatalf("it-ops missing %s", m)
		}
	}
	for _, m := range []Module{ModulePAM, ModuleObserve, ModuleNational} {
		if g.Allow(m) {
			t.Fatalf("it-ops must not include %s", m)
		}
	}
}

func TestBundleUnifiedIncludesObserve(t *testing.T) {
	g := FromModules([]Module{ModuleVuln, ModuleControlAI, ModuleResponse, ModuleManage, ModuleService, ModuleObserve})
	if !g.Allow(ModuleObserve) {
		t.Fatal("unified bundle needs observe")
	}
}

func TestBundleFullCatalog(t *testing.T) {
	g := FromModules([]Module{ModuleVuln, ModuleControlAI, ModuleResponse, ModuleManage, ModuleService, ModuleProvision, ModulePAM, ModuleObserve})
	for _, m := range []Module{ModuleManage, ModuleService, ModuleProvision, ModulePAM, ModuleObserve} {
		if !g.Allow(m) {
			t.Fatalf("full bundle missing %s", m)
		}
	}
}

func TestKnownModulesComplete(t *testing.T) {
	want := map[Module]bool{
		ModuleVuln: true, ModuleControlAI: true, ModuleResponse: true,
		ModuleManage: true, ModuleService: true, ModuleProvision: true,
		ModulePAM: true, ModuleObserve: true,
		ModuleFederated: true, ModuleNational: true,
	}
	for _, m := range KnownModules {
		if !want[m] {
			t.Fatalf("unexpected KnownModules entry %s", m)
		}
		delete(want, m)
	}
	if len(want) != 0 {
		t.Fatalf("KnownModules missing: %v", want)
	}
}
