package models

// Template описывает корневую структуру упрощенного YAML-шаблона сканирования.
type Template struct {
	ID       string    `yaml:"id" json:"id"`
	Info     Info      `yaml:"info" json:"info"`
	Requests []Request `yaml:"requests" json:"requests"`
}

// Info содержит метаданные шаблона сканирования.
type Info struct {
	Name        string `yaml:"name" json:"name"`
	Author      string `yaml:"author" json:"author"`
	Severity    string `yaml:"severity" json:"severity"`
	Description string `yaml:"description" json:"description"`
}

// Request описывает HTTP-запрос, который нужно выполнить для проверки.
type Request struct {
	Method   string            `yaml:"method" json:"method"`
	Path     []string          `yaml:"path" json:"path"`
	Headers  map[string]string `yaml:"headers" json:"headers"`
	Matchers []Matcher         `yaml:"matchers" json:"matchers"`
}

// Matcher описывает правило проверки ответа на наличие признаков уязвимости.
type Matcher struct {
	Type      string   `yaml:"type" json:"type"`
	Part      string   `yaml:"part" json:"part"`
	Words     []string `yaml:"words" json:"words"`
	Regex     []string `yaml:"regex" json:"regex"`
	Status    []int    `yaml:"status" json:"status"`
	Condition string   `yaml:"condition" json:"condition"`
}
