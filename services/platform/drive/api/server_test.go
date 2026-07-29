package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"era/services/platform/drive"
	driveapi "era/services/platform/drive/api"
	"era/services/platform/licensegate"

	"github.com/golang-jwt/jwt/v5"
)

func TestDriveAPIUploadListDownload(t *testing.T) {
	store := drive.NewMemoryStore()
	blobs := drive.NewMemoryBlobStore()
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive})
	secret := []byte("test-secret")
	srv := driveapi.NewServer(driveapi.Config{
		Store:     store,
		Blobs:     blobs,
		Gate:      gate,
		JWTSecret: secret,
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	tok, err := signTestToken(secret, "t1", "u@example.com", "u1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("hello drive"))
	_ = w.WriteField("name", "hello.txt")
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/objects", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload status %d: %s", rec.Code, rec.Body.String())
	}
	var obj drive.Object
	if err := json.NewDecoder(rec.Body).Decode(&obj); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drive/objects/"+obj.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("download status %d", rec.Code)
	}
	if got := rec.Body.String(); got != "hello drive" {
		t.Fatalf("download body %q", got)
	}
}

func TestDriveAPILicenseDenied(t *testing.T) {
	store := drive.NewMemoryStore()
	gate := licensegate.FromModules(nil)
	srv := driveapi.NewServer(driveapi.Config{
		Store:     store,
		Gate:      gate,
		JWTSecret: []byte("test-secret"),
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/folders", bytes.NewReader([]byte(`{"name":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ERA-Tenant", "t1")
	req.Header.Set("X-ERA-User", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestAttachmentLink(t *testing.T) {
	store := drive.NewMemoryStore()
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive})
	srv := driveapi.NewServer(driveapi.Config{
		Store:            store,
		Gate:             gate,
		WorkspaceBaseURL: "https://app.test.local",
		JWTSecret:        []byte("test-secret"),
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	ctx := httptest.NewRequest(http.MethodPost, "/", nil).Context()
	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID: "t1", Name: "a.bin", OwnerUserID: "u1", Data: []byte("x"),
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(`{"object_id":"` + obj.ID + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/links/attachment", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ERA-Tenant", "t1")
	req.Header.Set("X-ERA-User", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		URL string `json:"url"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.URL != "https://app.test.local/drive/o/"+obj.ID {
		t.Fatalf("url %q", resp.URL)
	}
}

func TestJWTPrincipal(t *testing.T) {
	secret := []byte("s")
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "u9", "tenant_id": "t9", "email": "u@x", "exp": time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	store := drive.NewMemoryStore()
	srv := driveapi.NewServer(driveapi.Config{
		Store: store, Gate: licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive}),
		JWTSecret: secret,
	})
	mux := http.NewServeMux()
	srv.Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/folders", stringsReader(`{"name":"A"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func signTestToken(secret []byte, tenantID, email, sub string, exp time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub": sub, "email": email, "tenant_id": tenantID, "exp": exp.Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

type stringsReader string

func (s stringsReader) Read(p []byte) (int, error) {
	return copy(p, []byte(s)), io.EOF
}
