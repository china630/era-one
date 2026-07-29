package drive_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/ui/drive"
)

func TestDriveSPAIndex(t *testing.T) {
	h := drive.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !contains(rec.Body.String(), "ERA Drive") {
		t.Fatal("missing title")
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
