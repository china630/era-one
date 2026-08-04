package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"era/services/comms/internal/httpauth"
	"era/services/comms/mail/internal/api"
	"era/services/comms/mail/internal/repo"
	"era/services/platform/licensegate"
)

func TestMailSendWithJWTNoDev(t *testing.T) {
	t.Setenv("ERA_MAIL_DEV", "")
	secret := "jwt-secret-for-oidc-test"
	t.Setenv("ERA_IDENTITY_JWT_SECRET", secret)
	t.Setenv("ERA_INTERNAL_TOKEN", "")

	m := repo.NewMemory()
	_, _ = m.CreateMailbox("t-demo", "alice@mail.gov.az", "pw", 10<<20)
	srv := api.NewServer(api.Config{
		Repo: m,
		Gate: licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsMailServer}),
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	tok, err := httpauth.MintDevJWT([]byte(secret), "t-demo", "alice", "user")
	if err != nil {
		t.Fatal(err)
	}
	body := `{"from":"alice@mail.gov.az","to":"bob@x.c","subject":"hi","body":"oidc"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mail/send", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Fatalf("JWT rejected: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Code >= 500 {
		t.Fatalf("server error: %d %s", rec.Code, rec.Body.String())
	}
}

func TestMailSendRejectsSpoofWithoutJWT(t *testing.T) {
	_ = os.Unsetenv("ERA_MAIL_DEV")
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "secret")
	m := repo.NewMemory()
	srv := api.NewServer(api.Config{Repo: m, Gate: licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsMailServer})})
	mux := http.NewServeMux()
	srv.Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mail/send", strings.NewReader(`{}`))
	req.Header.Set("X-ERA-Role", "admin")
	req.Header.Set("X-ERA-Tenant", "t-hack")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}
