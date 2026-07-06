package report

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRenderPDFValidHeader(t *testing.T) {
	now := time.Now().UTC()
	doc := DocumentFromCH("Bank", now.Add(-24*time.Hour), now)
	pdf := RenderPDF(doc)
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatal("missing PDF header")
	}
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Fatal("missing EOF marker")
	}
}

func TestExportZIPContainsManifest(t *testing.T) {
	now := time.Now().UTC()
	data, err := ExportZIP("Org", now.Add(-24*time.Hour), now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 100 {
		t.Fatal("zip too small")
	}
	if data[0] != 'P' || data[1] != 'K' {
		t.Fatal("not a zip file")
	}
}

func TestCHQueryStubWhenNoClient(t *testing.T) {
	m := QueryMetrics(nil, "x", time.Now(), time.Now())
	if m.TotalEvents != 125000 {
		t.Fatalf("stub total: got %d", m.TotalEvents)
	}
}

func TestRenderHTMLContainsMetrics(t *testing.T) {
	now := time.Now().UTC()
	doc := DocumentFromCH("Bank Demo", now.Add(-30*24*time.Hour), now)
	html := RenderHTML(doc)
	if !strings.Contains(html, "Bank Demo") {
		t.Fatal("missing org")
	}
	if !strings.Contains(html, "COMPLIANT") {
		t.Fatal("missing status")
	}
	if !strings.Contains(html, "125000") {
		t.Fatal("missing events from CH stub")
	}
}

func TestCHQueryStubDeterministic(t *testing.T) {
	m1 := CHQueryStub("x", time.Now(), time.Now())
	m2 := CHQueryStub("y", time.Now(), time.Now())
	if m1.TotalEvents != m2.TotalEvents {
		t.Fatal("stub should be deterministic")
	}
}
