package models

import "time"

// Finding представляет результат срабатывания правила сканирования.
type Finding struct {
	TemplateID        string    `yaml:"template_id" json:"template_id"`
	Target            string    `yaml:"target" json:"target"`
	Severity          string    `yaml:"severity" json:"severity"`
	VulnerabilityName string    `yaml:"vulnerability_name" json:"vulnerability_name"`
	MatchedURL        string    `yaml:"matched_url" json:"matched_url"`
	Timestamp         time.Time `yaml:"timestamp" json:"timestamp"`
}
