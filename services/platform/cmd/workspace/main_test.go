package main_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/workspace"
)

func TestWorkspacePackageLinked(t *testing.T) {
	srv := workspace.NewServer(workspace.Config{})
	mux := http.NewServeMux()
	srv.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}
