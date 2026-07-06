package cpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/assets/register" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"policy_version":"1.0"}`))
	}))
	defer srv.Close()
	c := New(srv.URL)
	if err := c.RegisterAsset("a1", "t1", "n1", "host1", "linux", "0.1.0"); err != nil {
		t.Fatal(err)
	}
}
