package license

import "fmt"

// Bundle — предустановленный набор модулей для коммерческой поставки (ADR-0005).
type Bundle string

const (
	BundleCoreAIResponse Bundle = "core-ai-response" // GA-1: Core + AI + Response
	BundleFullUpsell     Bundle = "full-upsell"        // vm + ai + response
)

// ModulesForBundle возвращает модули upsell для bundle (Core всегда включён).
func ModulesForBundle(b Bundle) ([]Module, error) {
	switch b {
	case BundleCoreAIResponse, "ga1":
		return []Module{ModuleControlAI, ModuleResponse}, nil
	case BundleFullUpsell:
		return []Module{ModuleVuln, ModuleControlAI, ModuleResponse}, nil
	case "":
		return nil, fmt.Errorf("empty bundle")
	default:
		return nil, fmt.Errorf("license: неизвестный bundle %q (доступны: core-ai-response, full-upsell)", b)
	}
}
