package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"era/services/control-plane/internal/store"
	"era/services/platform/licensegate"
)

func TestShellConfigOK(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "dev")
	s := New(store.NewMemory(), licensegate.DevDefault())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/shell/config", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("shell config %d %s", rec.Code, rec.Body.String())
	}
}

func TestEditionProxyViewerCannotMutate(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "dev")
	s := New(store.NewMemory(), licensegate.DevDefault())
	req := httptest.NewRequest(http.MethodPost, "/api/x/soar/api/v1/playbooks/isolate_host", nil)
	req.Header.Set("X-ERA-Role", "viewer")
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer mutate want 403 got %d", rec.Code)
	}
}

func TestAssetByID(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "dev")
	st := store.NewMemory()
	st.UpsertAsset(&store.Asset{NodeID: "n1", Hostname: "h1", TenantID: "default"})
	s := New(st, licensegate.DevDefault())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assets/n1", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("asset by id %d %s", rec.Code, rec.Body.String())
	}
}

func TestBYOConnectors(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "dev")
	s := New(store.NewMemory(), licensegate.DevDefault())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/byo/connectors", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("byo list %d", rec.Code)
	}
}

func TestControlRedirect(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "dev")
	s := New(store.NewMemory(), licensegate.DevDefault())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("redirect want 302 got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/ui/control/" {
		t.Fatalf("location %q", loc)
	}
}

func TestControlThemeTokens(t *testing.T) {
	css, err := os.ReadFile("../../../../ui/control-shell/web/control.css")
	if err != nil {
		t.Fatalf("read control.css: %v", err)
	}
	s := string(css)
	for _, want := range []string{
		`tokens/era-theme-control.css`,
		`tokens/era-tokens-base.css`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("control.css missing token import %q", want)
		}
	}
	theme, err := os.ReadFile("../../../../ui/shared-tokens/era-theme-control.css")
	if err != nil {
		t.Fatalf("read shared control theme: %v", err)
	}
	ts := string(theme)
	for _, want := range []string{
		"--era-accent",
		"--accent: var(--era-accent)",
		`--bg: var(--era-bg)`,
		`[data-line="control"]`,
	} {
		if !strings.Contains(ts, want) {
			t.Fatalf("era-theme-control.css missing %q", want)
		}
	}
	js, err := os.ReadFile("../../../../ui/control-shell/web/shell.js")
	if err != nil {
		t.Fatalf("read shell.js: %v", err)
	}
	if !strings.Contains(string(js), `setAttribute("data-line", "control")`) {
		t.Fatal("control shell.js must set data-line=control")
	}
}

func TestLegacyUIRedirects(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "dev")
	s := New(store.NewMemory(), licensegate.DevDefault())
	cases := []struct{ from, to string }{
		{"/ui/observe/", "/ui/control/observe/"},
		{"/ui/pam/", "/ui/control/pam/"},
		{"/ui/cases/", "/ui/control/workbench/"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.from, nil)
		rec := httptest.NewRecorder()
		s.Routes().ServeHTTP(rec, req)
		if rec.Code != http.StatusFound {
			t.Fatalf("%s want 302 got %d", c.from, rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != c.to {
			t.Fatalf("%s location want %q got %q", c.from, c.to, loc)
		}
	}
}
