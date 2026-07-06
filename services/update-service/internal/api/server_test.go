package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/update-service/internal/bundle"
)

func TestLatestBundle(t *testing.T) {
	priv, pub, err := bundle.LoadSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(priv, pub)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bundles/latest", nil)
	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}
