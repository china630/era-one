package docs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/ui/docs"
)

func TestDocsSPAIndex(t *testing.T) {
	h := docs.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "ERA Documents") {
		t.Fatal("missing title")
	}
	if !contains(body, "newDocBtn") && !contains(body, "New document") {
		t.Fatal("expected New document control")
	}
	for _, want := range []string{
		"boldBtn", "h1Btn", "listBtn", "importBtn", "exportBtn", "summarizeAIBtn",
		"alignJustifyBtn", "indentIncBtn", "formatPainterBtn", "superBtn",
		"undoBtn", "redoBtn", "docRuler", "tableInsertDlg", "wordCountDlg",
	} {
		if !contains(body, want) {
			t.Fatalf("missing toolbar control %q", want)
		}
	}
}

func TestDocsSPAFallbackDocPath(t *testing.T) {
	h := docs.Handler()
	req := httptest.NewRequest(http.MethodGet, "/doc-smoke-1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !contains(rec.Body.String(), "ERA Documents") {
		t.Fatal("expected index.html fallback for doc path")
	}
}

func TestDocsSPAAppCreateAndAuth(t *testing.T) {
	h := docs.Handler()
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "era_token") {
		t.Fatal("expected era_token auth")
	}
	if !contains(body, "POST") || !contains(body, "/api/v1/docs") {
		t.Fatal("expected create POST /api/v1/docs")
	}
	if !contains(body, "drive_object_id") {
		t.Fatal("expected redirect via drive_object_id")
	}
	if !contains(body, "verify-intent") || !contains(body, "verifyCommsIntent") {
		t.Fatal("expected AC-O8 VerifyIntent wire (verify-intent / verifyCommsIntent)")
	}
	if !contains(body, "intent_exp") || !contains(body, "intent_sig") {
		t.Fatal("expected intent_exp/intent_sig deep-link params")
	}
	if !contains(body, "set_inline_marks") {
		t.Fatal("expected set_inline_marks for bold/italic toolbar")
	}
	if !contains(body, "set_block_format") || !contains(body, "armFormatPainter") {
		t.Fatal("expected O-FMT-1 set_block_format / Format Painter")
	}
	if !contains(body, "initDocRuler") || !contains(body, "openWordCountDialog") || !contains(body, "openTableInsertDialog") {
		t.Fatal("expected O-FMT-2 ruler / word-count / table dialog")
	}
	if !contains(body, "/api/v1/docs/import") || !contains(body, "export/docx") {
		t.Fatal("expected import/export docx paths")
	}
	if !contains(body, "arrayBufferToBase64") {
		t.Fatal("expected chunked base64 import helper")
	}
}

func TestDocsSPAAppRemoteOps(t *testing.T) {
	h := docs.Handler()
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "applyRemoteOp") {
		t.Fatal("expected applyRemoteOp for peer live ops")
	}
	if !contains(body, "onSyncMessage") {
		t.Fatal("expected onSyncMessage WS handler")
	}
	if !contains(body, "ws.onmessage") {
		t.Fatal("expected ws.onmessage binding")
	}
	if !contains(body, "parsed.op") {
		t.Fatal("expected SyncEnvelope op unwrap")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
