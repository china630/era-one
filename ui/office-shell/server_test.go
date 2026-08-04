package officeshell

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHandlerServesOfficeCSS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/office.css", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`tokens/era-tokens-base.css`,
		`tokens/era-theme-sku.css`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing theme contract %q", want)
		}
	}
	req = httptest.NewRequest(http.MethodGet, "/tokens/era-theme-sku.css", nil)
	rec = httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("sku tokens status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `[data-sku="tables"]`) {
		t.Fatal("sku theme missing tables accent")
	}
}

func TestHandlerServesEraChrome(t *testing.T) {
	for _, path := range []string{"/era-chrome.css", "/era-chrome.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, rec.Code)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/era-chrome.js", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "EraChrome") {
		t.Fatal("era-chrome.js missing EraChrome")
	}
	if !strings.Contains(rec.Body.String(), "mountSwitcher") {
		t.Fatal("era-chrome.js missing mountSwitcher")
	}
}

func TestHandlerServesShellJS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/shell.js", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "EraOfficeShell") {
		t.Fatalf("missing EraOfficeShell: %q", rec.Body.String())
	}
}

func TestHandlerServesIconsJS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/icons.js", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "EraOfficeIcons") {
		t.Fatalf("missing EraOfficeIcons: %q", rec.Body.String())
	}
}

func TestLoginPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	LoginPage().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Sign in", "Create account", "login.js", "login.css"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in login page", want)
		}
	}
}

func TestHandlerServesLoginAssets(t *testing.T) {
	for _, path := range []string{"/login.css", "/login.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, rec.Code)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/login.js", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"inferProduct", "data-product", "ERA Communications"} {
		if !strings.Contains(body, want) {
			t.Fatalf("login.js missing product tint contract %q", want)
		}
	}
	req = httptest.NewRequest(http.MethodGet, "/login.css", nil)
	rec = httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	css := rec.Body.String()
	for _, want := range []string{`--login-accent: var(--era-accent)`, `data-product="comms"`, `--era-accent`} {
		if !strings.Contains(css, want) {
			t.Fatalf("login.css missing alias/tint contract %q", want)
		}
	}
}

func TestSKUThemeBodies(t *testing.T) {
	checks := []struct {
		rel  string
		sku  string
	}{
		{"../tables/web/index.html", `data-sku="tables"`},
		{"../docs/web/index.html", `data-sku="docs"`},
		{"../drive/web/index.html", `data-sku="drive"`},
		{"../presentations/web/index.html", `data-sku="pres"`},
		{"../projects/web/index.html", `data-sku="projects"`},
		{"../office-ai/web/index.html", `data-sku="ai"`},
	}
	for _, c := range checks {
		b, err := os.ReadFile(c.rel)
		if err != nil {
			t.Fatalf("read %s: %v", c.rel, err)
		}
		s := string(b)
		if !strings.Contains(s, `data-line="office"`) || !strings.Contains(s, c.sku) {
			t.Fatalf("%s missing theme attrs (want data-line=office and %s)", c.rel, c.sku)
		}
	}
}

