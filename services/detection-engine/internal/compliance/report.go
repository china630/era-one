// Package compliance — regulatory report helpers (ADR-0006 G-04).
package compliance

import "fmt"

// ReportLine — строка отчёта для регулятора.
type ReportLine struct {
	Org    string `json:"org"`
	Period string `json:"period"`
	Metric string `json:"metric"`
	Value  int    `json:"value"`
}

func BuildReport(org, period string, events, cases int) []ReportLine {
	return []ReportLine{
		{Org: org, Period: period, Metric: "events_ingested", Value: events},
		{Org: org, Period: period, Metric: "cases_opened", Value: cases},
		{Org: org, Period: period, Metric: "summary", Value: events + cases},
	}
}

func Summary(org, period string, events, cases int) string {
	return fmt.Sprintf("%s/%s events=%d cases=%d", org, period, events, cases)
}
