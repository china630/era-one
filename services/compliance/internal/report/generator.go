// Package report — регуляторная отчётность АЗ/ЦБ (F4-5).
package report

import (
	"fmt"
	"time"
)

type Input struct {
	OrgName       string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	TotalEvents   int64
	Detections    int64
	CriticalCount int64
	AssetsCovered float64
	PIILeaks      int64
}

type Document struct {
	TemplateID   string    `json:"template_id"`
	Regulator    string    `json:"regulator"`
	Language     string    `json:"language"`
	GeneratedAt  time.Time `json:"generated_at"`
	Organization string    `json:"organization"`
	Period       string    `json:"period"`
	Summary      Summary   `json:"summary"`
	Sections     []Section `json:"sections"`
}

type Summary struct {
	TotalEvents      int64   `json:"total_events"`
	Detections       int64   `json:"detections"`
	CriticalAlerts   int64   `json:"critical_alerts"`
	AssetCoveragePct float64 `json:"asset_coverage_pct"`
	PIIViolations    int64   `json:"pii_violations"`
	ComplianceStatus string  `json:"compliance_status"`
}

type Section struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func GenerateAZCB(in Input) Document {
	status := "COMPLIANT"
	if in.PIILeaks > 0 {
		status = "NON_COMPLIANT"
	}
	period := fmt.Sprintf("%s — %s",
		in.PeriodStart.Format("2006-01-02"),
		in.PeriodEnd.Format("2006-01-02"),
	)
	return Document{
		TemplateID:   "era-reg-az-cb-v1",
		Regulator:    "Central Bank of Azerbaijan / AZ Cybersecurity",
		Language:     "az",
		GeneratedAt:  time.Now().UTC(),
		Organization: in.OrgName,
		Period:       period,
		Summary: Summary{
			TotalEvents: in.TotalEvents, Detections: in.Detections,
			CriticalAlerts: in.CriticalCount, AssetCoveragePct: in.AssetsCovered * 100,
			PIIViolations: in.PIILeaks, ComplianceStatus: status,
		},
		Sections: []Section{
			{ID: "1", Title: "Ümumi baxış", Body: "ERA XDR platforması üzrə dövr yekunu."},
			{ID: "2", Title: "Hadisələr və deteksiyalar", Body: fmt.Sprintf("Hadisələr: %d; deteksiyalar: %d.", in.TotalEvents, in.Detections)},
			{ID: "3", Title: "PII və air-gap", Body: fmt.Sprintf("PII sızıntıları: %d. Xam Envelope export: qadağan.", in.PIILeaks)},
			{ID: "4", Title: "Aktivlər", Body: fmt.Sprintf("İnventar əhatəsi: %.0f%%.", in.AssetsCovered*100)},
		},
	}
}
