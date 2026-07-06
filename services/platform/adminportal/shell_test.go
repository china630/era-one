package adminportal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShellProductsAPI(t *testing.T) {
	sh := NewShell()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	sh.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Brand    string    `json:"brand"`
		Products []Product `json:"products"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Brand != "ERA One" || len(body.Products) != 3 {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestShellHealthz(t *testing.T) {
	sh := NewShell()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	sh.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestRegisterProduct(t *testing.T) {
	sh := NewShell()
	sh.Register(Product{Key: "era-control", Title: "ERA Control X", Status: StatusGA, SitePath: "/secure", AdminPath: "/admin/control"})
	list := sh.List()
	for _, p := range list {
		if p.Key == "era-control" && p.Title != "ERA Control X" {
			t.Fatalf("register failed: %+v", p)
		}
	}
}
