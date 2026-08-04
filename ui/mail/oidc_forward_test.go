package mail

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// G2: BFF forwards Bearer JWT to mail-api (no silent X-ERA-only path).
func TestProxySendForwardsBearer(t *testing.T) {
	secret := []byte("dev-only-change-in-prod")
	tok, err := SignTestToken(secret, "t-demo", "alice@mail.gov.az", "u-alice", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	s := NewServer(nil)
	s.MailAPIURL = upstream.URL
	s.JWTSecret = secret
	mux := http.NewServeMux()
	s.Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mail/api/send", strings.NewReader(`{"to":"b@x.c","subject":"s","body":"b"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Fatal("BFF rejected JWT")
	}
	if gotAuth != "Bearer "+tok {
		t.Fatalf("mail-api auth=%q want Bearer JWT", gotAuth)
	}
}
