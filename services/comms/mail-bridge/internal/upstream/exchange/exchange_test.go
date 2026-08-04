package exchange

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProxyEWSPassthroughGolden(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "exchange_findfolder_response.golden.xml"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "FindFolder") {
			t.Errorf("missing FindFolder in body")
		}
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write(want)
	}))
	defer srv.Close()

	be := New(srv.URL)
	status, resp, err := be.ProxyEWS(context.Background(), "FindFolder", []byte(`<FindFolder/>`), http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if strings.TrimSpace(string(resp)) != strings.TrimSpace(string(want)) {
		t.Fatalf("golden mismatch")
	}
}
