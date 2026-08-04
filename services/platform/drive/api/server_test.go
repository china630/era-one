package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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
	secret := []byte("test-secret")
	srv := driveapi.NewServer(driveapi.Config{
		Store:     store,
		Gate:      gate,
		JWTSecret: secret,
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	tok, _ := signTestToken(secret, "t1", "u@x", "u1", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/folders", bytes.NewReader([]byte(`{"name":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestSpoofHeadersRejected(t *testing.T) {
	store := drive.NewMemoryStore()
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive})
	srv := driveapi.NewServer(driveapi.Config{
		Store:     store,
		Gate:      gate,
		JWTSecret: []byte("test-secret"),
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/folders", bytes.NewReader([]byte(`{"name":"spoof"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ERA-Tenant", "evil-tenant")
	req.Header.Set("X-ERA-User", "evil-user")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for spoof-only headers, got %d", rec.Code)
	}
}

func TestJWTRequired(t *testing.T) {
	store := drive.NewMemoryStore()
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive})
	srv := driveapi.NewServer(driveapi.Config{
		Store:     store,
		Gate:      gate,
		JWTSecret: []byte("test-secret"),
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/folders", bytes.NewReader([]byte(`{"name":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rec.Code)
	}
}

func TestServiceTokenActingAs(t *testing.T) {
	store := drive.NewMemoryStore()
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive})
	srv := driveapi.NewServer(driveapi.Config{
		Store:        store,
		Gate:         gate,
		JWTSecret:    []byte("test-secret"),
		ServiceToken: "svc-secret",
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/folders", bytes.NewReader([]byte(`{"name":"svc"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer svc-secret")
	req.Header.Set("X-ERA-Tenant", "t-svc")
	req.Header.Set("X-ERA-User", "engine")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("service token status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDriveAPISearch(t *testing.T) {
	store := drive.NewMemoryStore()
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive})
	secret := []byte("test-secret")
	srv := driveapi.NewServer(driveapi.Config{
		Store:     store,
		Gate:      gate,
		JWTSecret: secret,
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	ctx := context.Background()
	folder, err := store.CreateFolder(ctx, "t1", "", "Shared Docs", "u1")
	if err != nil {
		t.Fatal(err)
	}
	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID: "t1", FolderID: folder.ID, Name: "doc-notes.md",
		OwnerUserID: "u1", Data: []byte("hi"),
	}, "k")
	if err != nil {
		t.Fatal(err)
	}

	tok, err := signTestToken(secret, "t1", "u@x", "u1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/drive/search?q=doc", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Folders []drive.Folder `json:"folders"`
		Objects []drive.Object `json:"objects"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Folders) != 1 || resp.Folders[0].ID != folder.ID {
		t.Fatalf("folders: %+v", resp.Folders)
	}
	if len(resp.Objects) != 1 || resp.Objects[0].ID != obj.ID {
		t.Fatalf("objects: %+v", resp.Objects)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drive/search?q=doc", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rec.Code)
	}
}

func TestAttachmentLink(t *testing.T) {
	store := drive.NewMemoryStore()
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive})
	secret := []byte("test-secret")
	srv := driveapi.NewServer(driveapi.Config{
		Store:            store,
		Gate:             gate,
		WorkspaceBaseURL: "https://app.test.local",
		JWTSecret:        secret,
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

	tok, _ := signTestToken(secret, "t1", "u@x", "u1", time.Now().Add(time.Hour))
	body := bytes.NewBufferString(`{"object_id":"` + obj.ID + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/links/attachment", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
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

func TestLockObjectPatch(t *testing.T) {
	secret := []byte("s")
	tok, err := signTestToken(secret, "t1", "o@x", "u-owner", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	store := drive.NewMemoryStore()
	obj, err := store.CreateObject(context.Background(), drive.CreateObjectInput{
		TenantID: "t1", Name: "lock-me.txt", OwnerUserID: "u-owner", Data: []byte("hi"),
	}, "k")
	if err != nil {
		t.Fatal(err)
	}
	srv := driveapi.NewServer(driveapi.Config{
		Store: store, Gate: licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive}),
		JWTSecret: secret,
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/drive/objects/"+obj.ID, stringsReader(`{"locked":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("lock status %d body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"locked_by":"u-owner"`) {
		t.Fatalf("lock body %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drive/objects/"+obj.ID+"/meta", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("meta status %d body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"locked_by":"u-owner"`) {
		t.Fatalf("meta missing lock: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/drive/objects/"+obj.ID, stringsReader(`{"locked":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unlock status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestRenameObjectAndGetMeta(t *testing.T) {
	secret := []byte("s")
	tok, err := signTestToken(secret, "t1", "o@x", "u-owner", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	store := drive.NewMemoryStore()
	obj, err := store.CreateObject(context.Background(), drive.CreateObjectInput{
		TenantID: "t1", Name: "old.txt", OwnerUserID: "u-owner", Data: []byte("hi"),
	}, "k")
	if err != nil {
		t.Fatal(err)
	}
	srv := driveapi.NewServer(driveapi.Config{
		Store: store, Gate: licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive}),
		JWTSecret: secret,
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/drive/objects/"+obj.ID, stringsReader(`{"name":"new.txt"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status %d body %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drive/objects/"+obj.ID+"/meta", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("meta status %d body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"name":"new.txt"`) {
		t.Fatalf("meta body %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"acl"`) {
		t.Fatalf("meta missing acl: %s", rec.Body.String())
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

func TestDriveAPIPutVersionStableID(t *testing.T) {
	store := drive.NewMemoryStore()
	blobs := drive.NewMemoryBlobStore()
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive})
	secret := []byte("test-secret")
	srv := driveapi.NewServer(driveapi.Config{
		Store: store, Blobs: blobs, Gate: gate, JWTSecret: secret,
	})
	mux := http.NewServeMux()
	srv.Register(mux)
	tok, err := signTestToken(secret, "t1", "u@example.com", "u1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("file", "doc.erad")
	_, _ = part.Write([]byte(`{"blocks":[]}`))
	_ = w.WriteField("name", "doc.erad")
	_ = w.WriteField("content_type", "application/vnd.era.erad")
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drive/objects", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create %d: %s", rec.Code, rec.Body.String())
	}
	var obj drive.Object
	if err := json.NewDecoder(rec.Body).Decode(&obj); err != nil {
		t.Fatal(err)
	}

	var body2 bytes.Buffer
	w2 := multipart.NewWriter(&body2)
	part2, _ := w2.CreateFormFile("file", "doc.erad")
	_, _ = part2.Write([]byte(`{"blocks":[{"t":"hi"}]}`))
	_ = w2.WriteField("content_type", "application/vnd.era.erad")
	_ = w2.Close()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/drive/objects/"+obj.ID+"/versions", &body2)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put version %d: %s", rec.Code, rec.Body.String())
	}
	var v2 drive.Object
	if err := json.NewDecoder(rec.Body).Decode(&v2); err != nil {
		t.Fatal(err)
	}
	if v2.ID != obj.ID || v2.Version != 2 {
		t.Fatalf("want id=%s ver=2 got id=%s ver=%d", obj.ID, v2.ID, v2.Version)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drive/objects/"+obj.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("download %d", rec.Code)
	}
	if got := rec.Body.String(); got != `{"blocks":[{"t":"hi"}]}` {
		t.Fatalf("download body %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/drive/objects/"+obj.ID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list versions %d", rec.Code)
	}
	var listed struct {
		Versions []drive.Version `json:"versions"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Versions) != 2 {
		t.Fatalf("want 2 versions, got %+v", listed.Versions)
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
