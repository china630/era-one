package drive_test

import (
	"context"
	"testing"

	"era/services/platform/drive"
)

func TestACLEnforcement(t *testing.T) {
	ctx := context.Background()
	store := drive.NewMemoryStore()
	blobs := drive.NewMemoryBlobStore()

	owner := drive.Principal{TenantID: "t1", UserID: "u-owner"}
	other := drive.Principal{TenantID: "t1", UserID: "u-other"}
	reader := drive.Principal{TenantID: "t1", UserID: "u-reader", Groups: []string{"grp-a"}}

	data := []byte("secret payload")
	key, err := blobs.Put(data)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID:    "t1",
		Name:        "report.pdf",
		ContentType: "application/pdf",
		OwnerUserID: owner.UserID,
		Data:        data,
	}, key)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetObject(ctx, "t1", obj.ID, owner); err != nil {
		t.Fatalf("owner read: %v", err)
	}
	if _, err := store.GetObject(ctx, "t1", obj.ID, other); err != drive.ErrForbidden {
		t.Fatalf("other user expected forbidden, got %v", err)
	}

	if err := store.UpdateACL(ctx, "t1", obj.ID, owner, []drive.ACLEntry{
		{Principal: "user:u-owner", Role: drive.ACLRoleOwner},
		{Principal: "group:grp-a", Role: drive.ACLRoleRead},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetObject(ctx, "t1", obj.ID, reader); err != nil {
		t.Fatalf("group reader: %v", err)
	}

	folder, err := store.CreateFolder(ctx, "t1", "", "Docs", owner.UserID)
	if err != nil {
		t.Fatal(err)
	}
	_, objs, err := store.ListChildren(ctx, "t1", folder.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 0 {
		t.Fatalf("expected empty folder, got %d objects", len(objs))
	}

	vs, err := store.ListVersions(ctx, "t1", obj.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].Version != 1 {
		t.Fatalf("versions: %+v", vs)
	}
}

func TestFolderDuplicateName(t *testing.T) {
	ctx := context.Background()
	store := drive.NewMemoryStore()
	if _, err := store.CreateFolder(ctx, "t1", "", "Inbox", "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateFolder(ctx, "t1", "", "Inbox", "u1"); err != drive.ErrDuplicate {
		t.Fatalf("expected duplicate, got %v", err)
	}
}
