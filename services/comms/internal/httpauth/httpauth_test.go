package httpauth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/internal/httpauth"
)

func TestUnauthWithoutToken(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "test-secret-32bytes-minimum!!")
	t.Setenv("ERA_INTERNAL_TOKEN", "")
	cfg := httpauth.FromEnv("ERA_MAIL_DEV", "mail.user")
	h := cfg.Wrap(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/api/v1/mail/send", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestSpoofHeadersRejectedWithoutDev(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "test-secret-32bytes-minimum!!")
	cfg := httpauth.FromEnv("ERA_MAIL_DEV", "mail.user")
	h := cfg.Wrap(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-ERA-Tenant", "evil")
	req.Header.Set("X-ERA-Role", "mail.user")
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("spoof want 401, got %d", rec.Code)
	}
}

func TestJWTOK(t *testing.T) {
	secret := []byte("test-secret-32bytes-minimum!!")
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", string(secret))
	tok, err := httpauth.MintDevJWT(secret, "t-demo", "alice", "mail.user")
	if err != nil {
		t.Fatal(err)
	}
	cfg := httpauth.FromEnv("ERA_MAIL_DEV", "mail.user")
	h := cfg.Wrap(func(w http.ResponseWriter, r *http.Request) {
		p, ok := httpauth.FromContext(r.Context())
		if !ok || p.Mode != "jwt" {
			t.Fatalf("principal %+v", p)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestInternalToken(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_INTERNAL_TOKEN", "int-secret")
	cfg := httpauth.FromEnv("ERA_MAIL_DEV", "")
	h := cfg.RequireInternal(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/audit", nil)
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("unauth want 403, got %d", rec.Code)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/internal/v1/audit", nil)
	req2.Header.Set("X-ERA-Internal-Token", "int-secret")
	rec2 := httptest.NewRecorder()
	h(rec2, req2)
	if rec2.Code != http.StatusNoContent {
		t.Fatalf("token want 204, got %d", rec2.Code)
	}
}

func TestDevBypass(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "1")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "")
	cfg := httpauth.FromEnv("ERA_MAIL_DEV", "")
	h := cfg.Wrap(func(w http.ResponseWriter, r *http.Request) {
		p, _ := httpauth.FromContext(r.Context())
		if p.Mode != "dev" {
			t.Fatalf("mode %s", p.Mode)
		}
		w.WriteHeader(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestBasicAuthDev(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "1")
	t.Setenv("ERA_PROTOCOL_BASIC_PASSWORD", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "")
	cfg := httpauth.FromEnv("ERA_MAIL_DEV", "")
	h := cfg.Wrap(func(w http.ResponseWriter, r *http.Request) {
		p, _ := httpauth.FromContext(r.Context())
		if p.Mode != "basic" || p.UserID != "alice@mail.gov.az" {
			t.Fatalf("principal %+v", p)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/caldav/", nil)
	req.SetBasicAuth("alice@mail.gov.az", "any")
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestBasicAuthPassword(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_PROTOCOL_BASIC_PASSWORD", "lab-proto-pass")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "secret")
	cfg := httpauth.FromEnv("ERA_MAIL_DEV", "")
	h := cfg.Wrap(func(w http.ResponseWriter, r *http.Request) {
		p, _ := httpauth.FromContext(r.Context())
		if p.Mode != "basic" {
			t.Fatalf("mode %s", p.Mode)
		}
		w.WriteHeader(http.StatusOK)
	})
	bad := httptest.NewRequest(http.MethodGet, "/", nil)
	bad.SetBasicAuth("alice", "wrong")
	recBad := httptest.NewRecorder()
	h(recBad, bad)
	if recBad.Code != http.StatusUnauthorized {
		t.Fatalf("wrong pass want 401, got %d", recBad.Code)
	}
	okReq := httptest.NewRequest(http.MethodGet, "/", nil)
	okReq.SetBasicAuth("alice", "lab-proto-pass")
	okReq.Header.Set("X-ERA-Tenant", "t-lab")
	recOK := httptest.NewRecorder()
	h(recOK, okReq)
	if recOK.Code != http.StatusOK {
		t.Fatalf("got %d", recOK.Code)
	}
}
