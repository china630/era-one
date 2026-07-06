package licensegate

import "os"

// FromEnv строит gate из ERA_LICENSE_MODULES (comma-separated).
// По умолчанию — DevDefault (federated выключен).
func FromEnv() *Gate {
	raw := os.Getenv("ERA_LICENSE_MODULES")
	if raw == "" {
		return DevDefault()
	}
	var mods []Module
	for _, part := range splitComma(raw) {
		mods = append(mods, Module(part))
	}
	return FromModules(mods)
}

func splitComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			p := s[start:i]
			if p != "" {
				parts = append(parts, p)
			}
			start = i + 1
		}
	}
	return parts
}
