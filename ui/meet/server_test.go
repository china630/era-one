package meet

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMeetUIShell(t *testing.T) {
	mux := http.NewServeMux()
	NewServer().Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/meet", nil)
	req.Header.Set("X-ERA-Tenant", "t-demo")
	req.Header.Set("X-ERA-Role", "vcs.user")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
