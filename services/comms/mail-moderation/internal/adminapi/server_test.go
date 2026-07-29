package adminapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/comms/mail-moderation/internal/adminapi"
	"era/services/comms/mail-moderation/internal/audit"
	"era/services/comms/mail-moderation/internal/engine"
	"era/services/comms/mail-moderation/internal/hold"
	"era/services/comms/mail-moderation/internal/notify"
	"era/services/comms/mail-moderation/internal/policy"
	"era/services/comms/mail-moderation/internal/resolve"
)

func TestYAML_ImportExport(t *testing.T) {
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
	srv := adminapi.New(eng, tok)
	mux := http.NewServeMux()
	srv.Register(mux)

	yamlBody := []byte("rules:\n  - id: t1\n    priority: 1\n    conditions:\n      keywords: [x]\n    moderator:\n      mode: static\n      static: [a@b.c]\n")
	req := httptest.NewRequest(http.MethodPost, "/v1/moderation/rules/import", bytes.NewReader(yamlBody))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("import %d %s", rr.Code, rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/v1/moderation/rules/export", nil)
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
	tok := notify.NewTokens([]byte("t"))
	eng := &engine.Engine{
		Holds:  hold.NewStore(),
		Groups: engine.StaticGroups{},
		Local:  []string{"c.local"},
	}
	srv := adminapi.New(eng, tok)
	mux := http.NewServeMux()
	srv.Register(mux)
	body := bytes.NewBufferString(`{"sender":"alex@c.local","curator":"sergey@c.local"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/moderation/hr/novices", body)
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
	tok := notify.NewTokens([]byte("t"))
	eng := &engine.Engine{Holds: hold.NewStore()}
	srv := adminapi.New(eng, tok)
	mux := http.NewServeMux()
	srv.Register(mux)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ui/", nil))
	if rr.Code != 200 || !bytes.Contains(rr.Body.Bytes(), []byte("ERA Mail Moderation")) {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
}
