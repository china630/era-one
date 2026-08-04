package meet

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMeetUIShell(t *testing.T) {
	mux := http.NewServeMux()
	NewServer().Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/meet/healthz", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"mode":"stub"`) {
		t.Fatalf("healthz want mode=stub: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/meet", nil)
	req.Header.Set("X-ERA-Tenant", "t-demo")
	req.Header.Set("X-ERA-Role", "vcs.user")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMeetJoinAndStatic(t *testing.T) {
	mux := http.NewServeMux()
	NewServer().Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/meet/join", nil)
	req.Header.Set("X-ERA-Tenant", "t-demo")
	req.Header.Set("X-ERA-Role", "vcs.user")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("join expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "livekit-stub") {
		t.Fatalf("join page missing stub ref")
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/meet/static/livekit-stub.js", nil)
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("static expected 200, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "ERALiveKit") {
		t.Fatalf("stub js missing ERALiveKit")
	}
}
