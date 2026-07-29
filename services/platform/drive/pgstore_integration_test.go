//go:build integration

package drive_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"era/services/platform/drive"
)

func openPgStore(t *testing.T) (*drive.PgStore, func()) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("ERA_OFFICE_DATABASE_URL"))
	if dsn == "" {
		t.Skip("ERA_OFFICE_DATABASE_URL not set")
	}
	s, err := drive.OpenPostgres(dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	ctx := context.Background()
	if err := s.EnsureSchema(ctx); err != nil {
		s.Close()
		t.Fatalf("schema: %v", err)
	}
	return s, func() { _ = s.Close() }
}

func TestPgStoreRoundtrip(t *testing.T) {
	ctx := context.Background()
	store, closeFn := openPgStore(t)
	defer closeFn()

	blobs := drive.NewMemoryBlobStore()
	owner := drive.Principal{TenantID: "t-pg-" + t.Name(), UserID: "u-owner"}
	other := drive.Principal{TenantID: "t-pg-" + t.Name(), UserID: "u-other"}

	data := []byte("persistent payload " + t.Name())
	key, err := blobs.Put(data)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID:    owner.TenantID,
		Name:        "roundtrip.bin",
		ContentType: "application/octet-stream",
		OwnerUserID: owner.UserID,
		Data:        data,
	}, key)
	if err != nil {
		t.Fatal(err)
	}

	store.Close()

	store2, err := drive.OpenPostgres(strings.TrimSpace(os.Getenv("ERA_OFFICE_DATABASE_URL")))
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	got, blobData, err := store2.GetObjectData(ctx, owner.TenantID, obj.ID, owner)
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if got.BlobKey != key {
		t.Fatalf("blob_key: got %q want %q", got.BlobKey, key)
	}
	if len(blobData) == 0 {
		fetched, err := blobs.Get(got.BlobKey)
		if err != nil {
			t.Fatal(err)
		}
		blobData = fetched
	}
	if string(blobData) != string(data) {
		t.Fatalf("data mismatch")
	}
	if _, err := store2.GetObject(ctx, owner.TenantID, obj.ID, other); err != drive.ErrForbidden {
		t.Fatalf("expected forbidden for other user, got %v", err)
	}
}

func TestPgStoreACLEnforcement(t *testing.T) {
	ctx := context.Background()
	store, closeFn := openPgStore(t)
	defer closeFn()
	blobs := drive.NewMemoryBlobStore()

	owner := drive.Principal{TenantID: "t-acl-" + t.Name(), UserID: "u-owner"}
	reader := drive.Principal{TenantID: owner.TenantID, UserID: "u-reader", Groups: []string{"grp-a"}}

	data := []byte("secret")
	key, _ := blobs.Put(data)
	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID:    owner.TenantID,
		Name:        "secret.pdf",
		ContentType: "application/pdf",
		OwnerUserID: owner.UserID,
		Data:        data,
	}, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateACL(ctx, owner.TenantID, obj.ID, owner, []drive.ACLEntry{
		{Principal: "user:u-owner", Role: drive.ACLRoleOwner},
		{Principal: "group:grp-a", Role: drive.ACLRoleRead},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetObject(ctx, owner.TenantID, obj.ID, reader); err != nil {
		t.Fatalf("group reader: %v", err)
	}
}

func TestPgStoreFolderDuplicate(t *testing.T) {
	ctx := context.Background()
	store, closeFn := openPgStore(t)
	defer closeFn()
	tid := "t-dup-" + t.Name()
	if _, err := store.CreateFolder(ctx, tid, "", "Inbox", "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateFolder(ctx, tid, "", "Inbox", "u1"); err != drive.ErrDuplicate {
		t.Fatalf("expected duplicate, got %v", err)
	}
}

func TestOpenFromEnvMemoryFallback(t *testing.T) {
	t.Setenv("ERA_OFFICE_DATABASE_URL", "")
	store, closeFn, err := drive.OpenFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()
	if _, ok := store.(*drive.MemoryStore); !ok {
		t.Fatalf("expected memory store, got %T", store)
	}
}
