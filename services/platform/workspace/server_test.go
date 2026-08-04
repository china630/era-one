package workspace_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/workspace"
)

func TestWorkspaceHealthz(t *testing.T) {
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

func TestWorkspaceTablesStub(t *testing.T) {
	srv := workspace.NewServer(workspace.Config{})
	mux := http.NewServeMux()
	srv.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/tables", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !contains(rec.Body.String(), "P2 roadmap") {
		t.Fatal("expected stub text when TablesUI unset")
	}
}

func TestWorkspaceTablesUI(t *testing.T) {
	srv := workspace.NewServer(workspace.Config{
		TablesUI: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ERA Tables live"))
		}),
	})
	mux := http.NewServeMux()
	srv.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/tables", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("redirect status %d", rec.Code)
	}
}

func TestWorkspaceDriveMount(t *testing.T) {
	srv := workspace.NewServer(workspace.Config{
		DriveUI: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("drive-ui"))
		}),
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/drive", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("redirect status %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/drive/" {
		t.Fatalf("location %q", loc)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/drive/", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || !contains(rec2.Body.String(), "drive-ui") {
		t.Fatalf("drive mount: %d %q", rec2.Code, rec2.Body.String())
	}
}

func TestWorkspaceOfficeShellMount(t *testing.T) {
	srv := workspace.NewServer(workspace.Config{
		OfficeShellUI: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/css")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(":root{--era-accent:#0b5fff}"))
		}),
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/office-assets/office.css", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !contains(rec.Body.String(), "--era-accent") {
		t.Fatalf("office-assets mount: %d %q", rec.Code, rec.Body.String())
	}
}

func TestWorkspaceIdentityProxyRegistered(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/staging/token" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok"}`))
	}))
	t.Cleanup(backend.Close)

	srv := workspace.NewServer(workspace.Config{IdentityAPIURL: backend.URL})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/oauth2/staging/token", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proxy status %d body %s", rec.Code, rec.Body.String())
	}
	if !contains(rec.Body.String(), "access_token") {
		t.Fatal("expected proxied token response")
	}
}

func TestWorkspaceAPIAuthRequired(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(backend.Close)

	srv := workspace.NewServer(workspace.Config{
		DriveAPIURL: backend.URL,
		JWTSecret:   []byte("ws-secret"),
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/drive/objects/x", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without JWT, got %d", rec.Code)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
