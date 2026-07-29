package mail_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"era/services/platform/drive"
	driveapi "era/services/platform/drive/api"
	"era/services/platform/licensegate"
	"era/ui/mail"
)

func TestDriveClientCreateAttachmentLink(t *testing.T) {
	store := drive.NewMemoryStore()
	srv := driveapi.NewServer(driveapi.Config{
		Store:            store,
		Gate:             licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive}),
		WorkspaceBaseURL: "https://app.test.local",
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
	link, err := client.CreateAttachmentLink("t1", obj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(link, "/drive/o/"+obj.ID) {
		t.Fatalf("link %q", link)
	}
}

func TestDriveClientNil(t *testing.T) {
	var c *mail.HTTPDriveClient
	if _, err := c.CreateAttachmentLink("t", "o"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSendWithDriveObjectID(t *testing.T) {
	store := drive.NewMemoryStore()
	dsrv := driveapi.NewServer(driveapi.Config{
		Store:            store,
		Gate:             licensegate.FromModules([]licensegate.Module{licensegate.ModulePlatformDrive}),
		WorkspaceBaseURL: "https://app.test.local",
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

	mailSrv := mail.NewServer(mail.NewHTTPDriveClient(dapi.URL))
	mux := http.NewServeMux()
	mailSrv.Register(mux)

	// internal helper endpoint tested via Drive field directly
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
