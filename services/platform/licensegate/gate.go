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
	ModulePerimeter Module = "perimeter"
	ModuleResolve   Module = "resolve"
	ModuleFederated Module = "federated"
	ModuleNational  Module = "national"
	ModuleCommsMailServer Module = "comms-mail-server"
	ModuleCommsMailConnect  Module = "comms-mail-connect"
	ModuleCommsMigration      Module = "comms-migration"
	ModuleCommsOutlookBridge  Module = "comms-outlook-bridge"
	ModuleCommsMailModeration Module = "comms-mail-moderation"
	ModuleCommsChat        Module = "comms-chat"
	ModuleCommsConference  Module = "comms-conference"
	ModuleCommsAI          Module = "comms-ai"
	ModulePlatformDrive    Module = "platform-drive"
	ModuleOfficeDocuments  Module = "office-documents"
	ModuleOfficeTables         Module = "office-tables"
	ModuleOfficePresentations  Module = "office-presentations"
	ModuleOfficeProjects       Module = "office-projects"
	ModuleOfficeAI             Module = "office-ai"
)

// KnownModules — все опциональные модули.
var KnownModules = []Module{ModuleVuln, ModuleControlAI, ModuleResponse, ModuleManage, ModuleService, ModuleProvision, ModulePAM, ModuleObserve, ModulePerimeter, ModuleResolve, ModuleFederated, ModuleNational, ModuleCommsMailServer, ModuleCommsMailConnect, ModuleCommsMigration, ModuleCommsOutlookBridge, ModuleCommsMailModeration, ModuleCommsChat, ModuleCommsConference, ModuleCommsAI, ModulePlatformDrive, ModuleOfficeDocuments, ModuleOfficeTables, ModuleOfficePresentations, ModuleOfficeProjects, ModuleOfficeAI}

// Gate описывает, какие модули включены в текущей лицензии.
type Gate struct {
	enabled map[Module]bool
}

// DevDefault — стандартная dev-лицензия без federated/national (F3-6).
func DevDefault() *Gate {
	g := &Gate{enabled: make(map[Module]bool)}
	for _, m := range []Module{
		ModuleVuln, ModuleControlAI, ModuleResponse, ModuleManage, ModuleService, ModuleProvision,
		ModulePAM, ModuleObserve, ModulePerimeter, ModuleResolve,
	} {
		g.enabled[m] = true
	}
	return g
}

// OfficeDevGate — Office P0 dev profile (ERA_OFFICE_DEV): platform-drive enabled.
func OfficeDevGate() *Gate {
	g := DevDefault()
	g.enabled[ModulePlatformDrive] = true
	g.enabled[ModuleOfficeDocuments] = true
	g.enabled[ModuleOfficeTables] = true
	g.enabled[ModuleOfficePresentations] = true
	g.enabled[ModuleOfficeProjects] = true
	g.enabled[ModuleOfficeAI] = true
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
