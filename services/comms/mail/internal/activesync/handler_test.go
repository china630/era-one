package activesync_test

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"era/services/comms/mail/internal/activesync"
	"era/services/comms/mail/internal/repo"
)

func loadGoldenHex(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	hexStr := strings.TrimSpace(string(raw))
	hexStr = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' {
			return -1
		}
		return r
	}, hexStr)
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return b
}

func TestProvisionGolden(t *testing.T) {
	want := loadGoldenHex(t, "provision_response.golden.wbxml.hex")
	h := &activesync.Handler{Repo: repo.NewMemory()}
	req := httptest.NewRequest(http.MethodPost, "/Microsoft-Server-ActiveSync?Cmd=Provision", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if string(rec.Body.Bytes()) != string(want) {
		t.Fatalf("body=%x want=%x", rec.Body.Bytes(), want)
	}
}

func TestFolderSyncGolden(t *testing.T) {
	reqBody := loadGoldenHex(t, "foldersync_request.golden.wbxml.hex")
	want := loadGoldenHex(t, "foldersync_response.golden.wbxml.hex")
	r := repo.NewMemory()
	mb, _ := r.CreateMailbox("t1", "alice@mail.gov.az", "pw", 1<<20)
	h := &activesync.Handler{Repo: r}
	req := httptest.NewRequest(http.MethodPost, "/Microsoft-Server-ActiveSync?Cmd=FolderSync&User=alice@mail.gov.az&DeviceId=dev-1", strings.NewReader(string(reqBody)))
	req.Header.Set("X-MS-DeviceId", "dev-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if string(rec.Body.Bytes()) != string(want) {
		t.Fatalf("body=%x want=%x", rec.Body.Bytes(), want)
	}
	key, ok := r.GetEASSyncKey("dev-1", mb.ID, "0")
	if !ok || key != "1" {
		t.Fatalf("sync key %q ok=%v", key, ok)
	}
}

func TestSyncGolden(t *testing.T) {
	reqBody := loadGoldenHex(t, "sync_request.golden.wbxml.hex")
	want := loadGoldenHex(t, "sync_response.golden.wbxml.hex")
	r := repo.NewMemory()
	mb, _ := r.CreateMailbox("t1", "alice@mail.gov.az", "pw", 1<<20)
	_, _ = r.AddEWSMessage("alice@mail.gov.az", "Hello", "body")
	h := &activesync.Handler{Repo: r}
	req := httptest.NewRequest(http.MethodPost, "/Microsoft-Server-ActiveSync?Cmd=Sync&User=alice@mail.gov.az&DeviceId=dev-1", strings.NewReader(string(reqBody)))
	req.Header.Set("X-MS-DeviceId", "dev-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if string(rec.Body.Bytes()) != string(want) {
		t.Fatalf("body=%x want=%x", rec.Body.Bytes(), want)
	}
	key, ok := r.GetEASSyncKey("dev-1", mb.ID, "1")
	if !ok || key != "1" {
		t.Fatalf("folder sync key %q ok=%v", key, ok)
	}
}

func TestPingGolden(t *testing.T) {
	want := loadGoldenHex(t, "ping_response.golden.wbxml.hex")
	h := &activesync.Handler{Repo: repo.NewMemory()}
	req := httptest.NewRequest(http.MethodPost, "/Microsoft-Server-ActiveSync?Cmd=Ping", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if string(rec.Body.Bytes()) != string(want) {
		t.Fatalf("body=%x want=%x", rec.Body.Bytes(), want)
	}
}
