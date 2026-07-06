// Package licensegate — проверка активации upsell-модулей (ADR-0010, F2-9).
package licensegate

// Module — лицензируемый upsell-модуль (см. ADR-0005/0010).
type Module string

const (
	ModuleVuln      Module = "vm"
	ModuleControlAI Module = "control-ai"
	ModuleAILegacy  Module = "ai" // deprecated; см. Allow
	ModuleResponse  Module = "response"
	ModuleManage    Module = "manage"
	ModuleService   Module = "service"
	ModuleProvision Module = "provision"
	ModulePAM       Module = "pam"
	ModuleObserve   Module = "observe"
	ModuleFederated Module = "federated"
	ModuleNational  Module = "national"
)

// KnownModules — все опциональные модули.
var KnownModules = []Module{ModuleVuln, ModuleControlAI, ModuleResponse, ModuleManage, ModuleService, ModuleProvision, ModulePAM, ModuleObserve, ModuleFederated, ModuleNational}

// Gate описывает, какие модули включены в текущей лицензии.
type Gate struct {
	enabled map[Module]bool
}

// DevDefault — стандартная dev-лицензия без federated/national (F3-6).
func DevDefault() *Gate {
	g := &Gate{enabled: make(map[Module]bool)}
	for _, m := range []Module{ModuleVuln, ModuleControlAI, ModuleResponse, ModuleManage, ModuleService, ModuleProvision, ModulePAM, ModuleObserve} {
		g.enabled[m] = true
	}
	return g
}

// DevAllEnabled — все модули включены (явные federated/national тесты).
func DevAllEnabled() *Gate {
	g := &Gate{enabled: make(map[Module]bool)}
	for _, m := range KnownModules {
		g.enabled[m] = true
	}
	return g
}

// FromModules строит gate из списка модулей лицензии.
func FromModules(mods []Module) *Gate {
	g := &Gate{enabled: make(map[Module]bool)}
	for _, m := range mods {
		g.enabled[m] = true
	}
	return g
}

// Allow возвращает true, если модуль активирован.
func (g *Gate) Allow(mod Module) bool {
	if g == nil {
		return true
	}
	if g.enabled[mod] {
		return true
	}
	if mod == ModuleControlAI && g.enabled[ModuleAILegacy] {
		return true
	}
	return false
}
