package icewarp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxyEWSPassthrough(t *testing.T) {
	var gotAction string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAction = r.Header.Get("SOAPAction")
		if r.URL.Path != "/ews/Exchange.asmx" {
			t.Errorf("path %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "FindFolder") {
			t.Errorf("body missing FindFolder")
		}
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<soap:Envelope><soap:Body><FindFolderResponse/></soap:Body></soap:Envelope>`))
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
	if !strings.Contains(string(resp), "FindFolderResponse") {
		t.Fatalf("resp %s", resp)
	}
	if gotAction == "" {
		t.Fatal("missing SOAPAction upstream")
	}
}
