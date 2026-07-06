// HTML/PDF export и stub ClickHouse query (S6-4).
package report

import (
	"fmt"
	"strings"
	"time"
)

// CHMetrics — агрегаты из ClickHouse (stub или реальный клиент позже).
type CHMetrics struct {
	TotalEvents   int64
	Detections    int64
	CriticalCount int64
	AssetsCovered float64
	PIILeaks      int64
}

// CHQueryStub возвращает детерминированные метрики для отчёта без live CH.
func CHQueryStub(org string, periodStart, periodEnd time.Time) CHMetrics {
	_ = org
	_ = periodStart
	_ = periodEnd
	return CHMetrics{
		TotalEvents: 125000, Detections: 87, CriticalCount: 5,
		AssetsCovered: 0.93, PIILeaks: 0,
	}
}

// DocumentFromCH строит regulatory document из CH (ERA_CH_URL) или stub.
func DocumentFromCH(org string, periodStart, periodEnd time.Time) Document {
	return DocumentFromCHWithClient(NewCHClientFromEnv(), org, periodStart, periodEnd)
}

// DocumentFromCHWithClient — injectable клиент для тестов.
func DocumentFromCHWithClient(ch *CHClient, org string, periodStart, periodEnd time.Time) Document {
	m := QueryMetrics(ch, org, periodStart, periodEnd)
	return GenerateAZCB(Input{
		OrgName: org, PeriodStart: periodStart, PeriodEnd: periodEnd,
		TotalEvents: m.TotalEvents, Detections: m.Detections,
		CriticalCount: m.CriticalCount, AssetsCovered: m.AssetsCovered, PIILeaks: m.PIILeaks,
	})
}

// RenderHTML экспортирует Document в простой HTML (air-gap, без внешних CDN).
func RenderHTML(doc Document) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html lang=\"az\"><head><meta charset=\"utf-8\">")
	b.WriteString("<title>ERA XDR Regulatory Report</title>")
	b.WriteString("<style>body{font-family:sans-serif;margin:2em}h1{color:#1a365d}table{border-collapse:collapse}td,th{border:1px solid #ccc;padding:8px}</style>")
	b.WriteString("</head><body>")
	fmt.Fprintf(&b, "<h1>%s</h1>", escape(doc.Organization))
	fmt.Fprintf(&b, "<p><strong>Regulator:</strong> %s</p>", escape(doc.Regulator))
	fmt.Fprintf(&b, "<p><strong>Period:</strong> %s</p>", escape(doc.Period))
	fmt.Fprintf(&b, "<p><strong>Status:</strong> %s</p>", escape(doc.Summary.ComplianceStatus))
	b.WriteString("<table><tr><th>Metric</th><th>Value</th></tr>")
	fmt.Fprintf(&b, "<tr><td>Total events</td><td>%d</td></tr>", doc.Summary.TotalEvents)
	fmt.Fprintf(&b, "<tr><td>Detections</td><td>%d</td></tr>", doc.Summary.Detections)
	fmt.Fprintf(&b, "<tr><td>Critical alerts</td><td>%d</td></tr>", doc.Summary.CriticalAlerts)
	fmt.Fprintf(&b, "<tr><td>Asset coverage</td><td>%.0f%%</td></tr>", doc.Summary.AssetCoveragePct)
	fmt.Fprintf(&b, "<tr><td>PII violations</td><td>%d</td></tr>", doc.Summary.PIIViolations)
	b.WriteString("</table>")
	for _, sec := range doc.Sections {
		fmt.Fprintf(&b, "<h2>%s</h2><p>%s</p>", escape(sec.Title), escape(sec.Body))
	}
	b.WriteString("</body></html>")
	return b.String()
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}
