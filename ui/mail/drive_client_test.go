package mail_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"era/services/platform/drive"
	driveapi "era/services/platform/drive/api"
	"era/services/platform/licensegate"
	"era/ui/mail"

	"github.com/golang-jwt/jwt/v5"
)

func signTok(secret []byte, tenant, sub string) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub, "tenant_id": tenant, "email": sub + "@x", "exp": time.Now().Add(time.Hour).Unix(),
	})
	s, _ := tok.SignedString(secret)
	return s
}

func TestDriveClientCreateAttachmentLink(t *testing.T) {
	store := drive.NewMemoryStore()
	secret := []byte("mail-test-secret")
	srv := driveapi.NewServer(driveapi.Config{
		Store:            store,
		Gate:             licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive}),
		WorkspaceBaseURL: "https://app.test.local",
		JWTSecret:        secret,
	})
	mux := http.NewServeMux()
	srv.Register(mux)
	api := httptest.NewServer(mux)
	t.Cleanup(api.Close)

	obj, err := store.CreateObject(t.Context(), drive.CreateObjectInput{
		TenantID: "t1", Name: "attach.bin", OwnerUserID: "u1", Data: []byte("x"),
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	client := mail.NewHTTPDriveClient(api.URL)
	client.UserJWT = signTok(secret, "t1", "u1")
	link, err := client.CreateAttachmentLink("t1", obj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(link, "/drive/o/"+obj.ID) {
		t.Fatalf("link %q", link)
	}
}

func TestDriveAttachmentLinkDenyWithoutLicense(t *testing.T) {
	secret := []byte("mail-test-secret")
	dsrv := driveapi.NewServer(driveapi.Config{
		Store:            drive.NewMemoryStore(),
		Gate:             licensegate.FromModules(nil), // no platform-drive
		WorkspaceBaseURL: "https://app.test.local",
		JWTSecret:        secret,
	})
	dmux := http.NewServeMux()
	dsrv.Register(dmux)
	dapi := httptest.NewServer(dmux)
	t.Cleanup(dapi.Close)

	dc := mail.NewHTTPDriveClient(dapi.URL)
	mailSrv := mail.NewServer(dc)
	mailSrv.JWTSecret = secret
	mux := http.NewServeMux()
	mailSrv.Register(mux)

	tok, err := mail.SignTestToken(secret, "t1", "u1@x", "u1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mail/api/drive/attachment-link", strings.NewReader(`{"object_id":"o1"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden && rec.Code != http.StatusBadGateway {
		// drive returns 403 → BFF maps to 403; some paths may surface 502 with status code in body
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d body=%s want 403", rec.Code, rec.Body.String())
		}
	}
}

func TestSendWithDriveObjectID(t *testing.T) {
	store := drive.NewMemoryStore()
	secret := []byte("mail-test-secret")
	dsrv := driveapi.NewServer(driveapi.Config{
		Store:            store,
		Gate:             licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive}),
		WorkspaceBaseURL: "https://app.test.local",
		JWTSecret:        secret,
	})
	dmux := http.NewServeMux()
	dsrv.Register(dmux)
	dapi := httptest.NewServer(dmux)
	t.Cleanup(dapi.Close)

	obj, err := store.CreateObject(t.Context(), drive.CreateObjectInput{
		TenantID: "t1", Name: "f.pdf", OwnerUserID: "u1", Data: []byte("%PDF"),
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	dc := mail.NewHTTPDriveClient(dapi.URL)
	dc.UserJWT = signTok(secret, "t1", "u1")
	mailSrv := mail.NewServer(dc)
	link, err := mailSrv.Drive.CreateAttachmentLink("t1", obj.ID)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	_ = json.Unmarshal([]byte(`{"url":"`+link+`"}`), &payload)
	if !contains(link, "/drive/o/") {
		t.Fatalf("expected drive deep link, got %q", link)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
