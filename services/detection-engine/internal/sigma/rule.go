// Package sigma — упрощённый парсер и matcher Sigma-правил (Фаза 2).
package sigma

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule — минимальное подмножество Sigma для MVP.
type Rule struct {
	ID         string            `yaml:"id"`
	Title      string            `yaml:"title"`
	Level      string            `yaml:"level"`
	Logsource  map[string]string `yaml:"logsource"`
	Detection  map[string]any    `yaml:"detection"`
	Status     string            `yaml:"status"`
	filePath   string
}

// LoadDir загружает все .yml/.yaml из каталога.
func LoadDir(dir string) ([]*Rule, error) {
	var rules []*Rule
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var r Rule
		if err := yaml.Unmarshal(data, &r); err != nil {
			return err
		}
		if r.ID == "" || r.Title == "" {
			return nil
		}
		r.filePath = path
		rules = append(rules, &r)
		return nil
	})
	return rules, err
}

// Match проверяет событие (category + текст payload) против правила.
func (r *Rule) Match(category string, payloadText string) bool {
	if cat := r.Logsource["category"]; cat != "" && cat != category {
		return false
	}
	sel, ok := r.Detection["selection"].(map[string]any)
	if !ok {
		return false
	}
	for k, v := range sel {
		field, modifier := parseField(k)
		val := toString(v)
		haystack := payloadText
		if field != "" && !strings.Contains(strings.ToLower(haystack), strings.ToLower(field)) {
			// поле не указано явно в payload — ищем по всему тексту
		}
		switch modifier {
		case "contains":
			if !strings.Contains(strings.ToLower(haystack), strings.ToLower(val)) {
				return false
			}
		default:
			if !strings.Contains(strings.ToLower(haystack), strings.ToLower(val)) {
				return false
			}
		}
	}
	return true
}

func parseField(k string) (field, modifier string) {
	parts := strings.SplitN(k, "|", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return k, "contains"
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

// Lint проверяет корпус правил.
func Lint(rules []*Rule) []string {
	var errs []string
	seen := map[string]bool{}
	for _, r := range rules {
		if r.ID == "" {
			errs = append(errs, "missing id")
		}
		if seen[r.ID] {
			errs = append(errs, "duplicate id: "+r.ID)
		}
		seen[r.ID] = true
		if r.Detection == nil {
			errs = append(errs, r.ID+": missing detection")
		}
	}
	return errs
}
