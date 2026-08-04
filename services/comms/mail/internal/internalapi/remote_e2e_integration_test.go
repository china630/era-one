//go:build integration

package internalapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"era/services/comms/mail/internal/repo"
)

// Remote deliver/list round-trip via internal HTTP (R2-D).
func TestRemoteStoreE2E(t *testing.T) {
	dsn := os.Getenv("ERA_COMMS_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://era:era_ci_pw@127.0.0.1:5432/era_cp?sslmode=disable"
	}
	pg, err := repo.OpenPostgres(dsn)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(func() { _ = pg.Close() })

	email := "remote-e2e@mail.gov.az"
	_, _ = pg.CreateMailbox("t-demo", email, "remote-pass", 1<<30)

	mux := http.NewServeMux()
	h := &Handler{Repo: pg}
	h.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	raw := "From: alice@mail.gov.az\r\nTo: remote-e2e@mail.gov.az\r\nSubject: remote\r\n\r\nHello remote"
	deliverBody, _ := json.Marshal(map[string]string{
		"email": email,
		"from":  "alice@mail.gov.az",
		"raw":   raw,
	})
	resp, err := http.Post(srv.URL+"/internal/v1/mail/deliver", "application/json", bytes.NewReader(deliverBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("deliver status %d", resp.StatusCode)
	}

	listResp, err := http.Get(srv.URL + "/internal/v1/mail/list?email=" + email)
	if err != nil {
		t.Fatal(err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list status %d", listResp.StatusCode)
	}
	var list struct {
		Messages []struct {
			Raw string `json:"raw"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Messages) == 0 || !strings.Contains(list.Messages[0].Raw, "Hello remote") {
		t.Fatalf("expected message body in list, got %+v", list.Messages)
	}

	verifyBody, _ := json.Marshal(map[string]string{"email": email, "password": "remote-pass"})
	vresp, err := http.Post(srv.URL+"/internal/v1/auth/verify", "application/json", bytes.NewReader(verifyBody))
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		OK bool `json:"ok"`
	}
	_ = json.NewDecoder(vresp.Body).Decode(&v)
	if !v.OK {
		t.Fatal("verify password failed")
	}
}
