package adminapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/internal/httpauth"
	"era/services/comms/mail-moderation/internal/adminapi"
	"era/services/comms/mail-moderation/internal/audit"
	"era/services/comms/mail-moderation/internal/engine"
	"era/services/comms/mail-moderation/internal/hold"
	"era/services/comms/mail-moderation/internal/notify"
	"era/services/comms/mail-moderation/internal/policy"
	"era/services/comms/mail-moderation/internal/resolve"
)

func withAdminDEV(req *http.Request) {
	req.Header.Set("X-ERA-Tenant", "t-demo")
	req.Header.Set("X-ERA-Role", "mm.admin")
	req.Header.Set("X-ERA-User", "admin")
}

func testEngine() (*engine.Engine, *notify.Tokens) {
	tok := notify.NewTokens([]byte("t"))
	eng := &engine.Engine{
		Rules:    nil,
		Holds:    hold.NewStore(),
		Resolve:  &resolve.Resolver{Dir: &resolve.MemoryDir{}},
		Audit:    &audit.Memory{},
		Upstream: &engine.MemoryUpstream{},
		Notify:   &notify.Service{Mailer: &notify.Recorder{}, Tokens: tok, PublicBase: "http://localhost"},
		Groups:   engine.StaticGroups{},
		Local:    []string{"company.local"},
	}
	return eng, tok
}

func TestYAML_ImportExport(t *testing.T) {
	t.Setenv("ERA_MM_DEV", "1")
	eng, tok := testEngine()
	srv := adminapi.New(eng, tok)
	mux := http.NewServeMux()
	srv.Register(mux)

	yamlBody := []byte("rules:\n  - id: t1\n    priority: 1\n    conditions:\n      keywords: [x]\n    moderator:\n      mode: static\n      static: [a@b.c]\n")
	req := httptest.NewRequest(http.MethodPost, "/v1/moderation/rules/import", bytes.NewReader(yamlBody))
	withAdminDEV(req)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("import %d %s", rr.Code, rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/v1/moderation/rules/export", nil)
	withAdminDEV(req)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatal(rr.Code)
	}
	doc, err := policy.ParseDocument(rr.Body.Bytes())
	if err != nil || len(doc.Rules) != 1 || doc.Rules[0].ID != "t1" {
		t.Fatalf("%v %v", doc, err)
	}
}

func TestTemplates(t *testing.T) {
	tpl := adminapi.Templates()
	if len(tpl) != 4 {
		t.Fatalf("%d", len(tpl))
	}
	if _, ok := tpl["moderated-dl"]; !ok {
		t.Fatal("missing moderated-dl template")
	}
}

func TestHRNovices(t *testing.T) {
	t.Setenv("ERA_MM_DEV", "1")
	eng, tok := testEngine()
	eng.Local = []string{"c.local"}
	srv := adminapi.New(eng, tok)
	mux := http.NewServeMux()
	srv.Register(mux)
	body := bytes.NewBufferString(`{"sender":"alex@c.local","curator":"sergey@c.local"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/moderation/hr/novices", body)
	withAdminDEV(req)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	if srv.Novices["alex@c.local"] != "sergey@c.local" {
		t.Fatal(srv.Novices)
	}
	sg := eng.Groups.(engine.StaticGroups)
	if len(sg["alex@c.local"]) == 0 || sg["alex@c.local"][0] != "novices" {
		t.Fatal(sg)
	}
}

func TestUI(t *testing.T) {
	t.Setenv("ERA_MM_DEV", "1")
	eng, tok := testEngine()
	srv := adminapi.New(eng, tok)
	mux := http.NewServeMux()
	srv.Register(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	withAdminDEV(req)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 || !bytes.Contains(rr.Body.Bytes(), []byte("ERA Mail Moderation")) {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
}

func TestAdminUnauth(t *testing.T) {
	t.Setenv("ERA_MM_DEV", "")
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", "test-secret-32bytes-minimum!!")
	t.Setenv("ERA_INTERNAL_TOKEN", "")
	eng, tok := testEngine()
	srv := adminapi.New(eng, tok)
	mux := http.NewServeMux()
	srv.Register(mux)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/moderation/rules"},
		{http.MethodGet, "/v1/moderation/holds"},
		{http.MethodGet, "/v1/moderation/templates"},
		{http.MethodPost, "/v1/moderation/simulate"},
		{http.MethodGet, "/ui/"},
	}
	for _, p := range paths {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(p.method, p.path, nil))
		if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
			t.Fatalf("%s %s: want 401/403, got %d", p.method, p.path, rr.Code)
		}
	}
}

func TestForceReleaseRequiresMMAdmin(t *testing.T) {
	secret := []byte("test-secret-32bytes-minimum!!")
	t.Setenv("ERA_MM_DEV", "")
	t.Setenv("ERA_MAIL_DEV", "")
	t.Setenv("ERA_IDENTITY_JWT_SECRET", string(secret))

	eng, tok := testEngine()
	rec, err := eng.Holds.Put(hold.Record{Sender: "a@b.c", Subject: "m1", Raw: []byte("raw")})
	if err != nil {
		t.Fatal(err)
	}
	srv := adminapi.New(eng, tok)
	mux := http.NewServeMux()
	srv.Register(mux)

	// unauth
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/moderation/holds/"+rec.ID+"?action=approve", nil))
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Fatalf("unauth want 401/403, got %d", rr.Code)
	}

	// authenticated but not mm.admin
	userTok, err := httpauth.MintDevJWT(secret, "t-demo", "bob", "mail.user")
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/moderation/holds/"+rec.ID+"?action=approve", nil)
	req.Header.Set("Authorization", "Bearer "+userTok)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("non-admin want 403, got %d", rr.Code)
	}

	// mm.admin OK
	adminTok, err := httpauth.MintDevJWT(secret, "t-demo", "admin", "mm.admin")
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/moderation/holds/"+rec.ID+"?action=approve", nil)
	req.Header.Set("Authorization", "Bearer "+adminTok)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("mm.admin want 204, got %d %s", rr.Code, rr.Body.String())
	}
}
