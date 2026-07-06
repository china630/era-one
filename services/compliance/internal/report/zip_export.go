// ZIP-пакет регуляторного экспорта (S7-14).
package report

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// ExportZIP собирает regulatory pack: manifest + HTML + JSON.
func ExportZIP(org string, periodStart, periodEnd time.Time, ch *CHClient) ([]byte, error) {
	doc := DocumentFromCHWithClient(ch, org, periodStart, periodEnd)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	manifest := fmt.Sprintf("ERA XDR Regulatory Export\norg=%s\nperiod=%s\ngenerated=%s\n",
		org, doc.Period, doc.GeneratedAt.Format(time.RFC3339))
	if err := writeZipEntry(zw, "manifest.txt", []byte(manifest)); err != nil {
		return nil, err
	}
	if err := writeZipEntry(zw, "report.html", []byte(RenderHTML(doc))); err != nil {
		return nil, err
	}
	jb, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := writeZipEntry(zw, "report.json", jb); err != nil {
		return nil, err
	}
	if err := writeZipEntry(zw, "report.pdf", RenderPDF(doc)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
