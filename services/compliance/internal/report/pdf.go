// Минимальный PDF-экспорт (S6-4) — text/pdf stub без внешних зависимостей.
package report

import (
	"bytes"
	"fmt"
	"strings"
)

// RenderPDF возвращает минимальный валидный PDF 1.4 с текстом отчёта.
func RenderPDF(doc Document) []byte {
	text := pdfEscape(fmt.Sprintf(
		"ERA XDR Regulatory Report Org: %s Period: %s Status: %s Events: %d Detections: %d",
		doc.Organization, doc.Period, doc.Summary.ComplianceStatus,
		doc.Summary.TotalEvents, doc.Summary.Detections,
	))
	stream := fmt.Sprintf("BT /F1 11 Tf 50 750 Td (%s) Tj ET", text)
	parts := []string{
		"1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n",
		"2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n",
		"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj\n",
		fmt.Sprintf("4 0 obj<</Length %d>>stream\n%s\nendstream\nendobj\n", len(stream), stream),
		"5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\n",
	}

	var body bytes.Buffer
	body.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for _, p := range parts {
		offsets = append(offsets, body.Len())
		body.WriteString(p)
	}

	var xref bytes.Buffer
	xref.WriteString("xref\n")
	xref.WriteString(fmt.Sprintf("0 %d\n", len(offsets)))
	xref.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		xref.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	trailer := fmt.Sprintf("trailer<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF", len(offsets), body.Len()+xref.Len())

	return append(body.Bytes(), append(xref.Bytes(), []byte(trailer)...)...)
}

func pdfEscape(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\n", " ")
	return r.Replace(s)
}
