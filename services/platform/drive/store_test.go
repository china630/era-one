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

func TestPutVersionStableID(t *testing.T) {
	ctx := context.Background()
	store := drive.NewMemoryStore()
	owner := drive.Principal{TenantID: "t1", UserID: "u-owner"}
	other := drive.Principal{TenantID: "t1", UserID: "u-other"}

	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID: "t1", Name: "sheet.erat", ContentType: "application/vnd.era.erat",
		OwnerUserID: owner.UserID, Data: []byte(`{"v":1}`),
	}, "k-v1")
	if err != nil {
		t.Fatal(err)
	}
	if obj.Version != 1 {
		t.Fatalf("version want 1 got %d", obj.Version)
	}

	updated, err := store.PutVersion(ctx, "t1", obj.ID, owner, drive.PutVersionInput{
		Data:        []byte(`{"v":2}`),
		ContentType: "application/vnd.era.erat",
	}, "k-v2")
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != obj.ID {
		t.Fatalf("stable id changed: %s -> %s", obj.ID, updated.ID)
	}
	if updated.Version != 2 {
		t.Fatalf("version want 2 got %d", updated.Version)
	}
	got, data, err := store.GetObjectData(ctx, "t1", obj.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 2 || string(data) != `{"v":2}` {
		t.Fatalf("head content: ver=%d data=%q", got.Version, data)
	}
	vs, err := store.ListVersions(ctx, "t1", obj.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 2 || vs[0].Version != 1 || vs[1].Version != 2 {
		t.Fatalf("versions: %+v", vs)
	}
	if _, err := store.PutVersion(ctx, "t1", obj.ID, other, drive.PutVersionInput{
		Data: []byte("x"),
	}, "k-x"); err != drive.ErrForbidden {
		t.Fatalf("other put expected forbidden, got %v", err)
	}
}

func TestSearchByName(t *testing.T) {
	ctx := context.Background()
	store := drive.NewMemoryStore()
	owner := drive.Principal{TenantID: "t1", UserID: "u-owner"}
	other := drive.Principal{TenantID: "t1", UserID: "u-other"}

	folder, err := store.CreateFolder(ctx, "t1", "", "Reports Q2", owner.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateFolder(ctx, "t1", "", "Inbox", owner.UserID); err != nil {
		t.Fatal(err)
	}
	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID: "t1", FolderID: folder.ID, Name: "Quarterly Report.pdf",
		OwnerUserID: owner.UserID, Data: []byte("x"),
	}, "k1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID: "t1", Name: "notes.txt", OwnerUserID: owner.UserID, Data: []byte("y"),
	}, "k2"); err != nil {
		t.Fatal(err)
	}
	// Other tenant must not appear.
	if _, err := store.CreateFolder(ctx, "t2", "", "Reports Q2", "u2"); err != nil {
		t.Fatal(err)
	}

	folders, objects, err := store.Search(ctx, "t1", "report", owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].ID != folder.ID {
		t.Fatalf("folders: %+v", folders)
	}
	if len(objects) != 1 || objects[0].ID != obj.ID {
		t.Fatalf("objects: %+v", objects)
	}

	// ACL: non-owner cannot see private object; folder remains tenant-visible.
	folders, objects, err = store.Search(ctx, "t1", "report", other)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 {
		t.Fatalf("other folders: %+v", folders)
	}
	if len(objects) != 0 {
		t.Fatalf("other should not see private object, got %+v", objects)
	}

	folders, objects, err = store.Search(ctx, "t1", "   ", owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 0 || len(objects) != 0 {
		t.Fatalf("empty query should return nothing, got %d/%d", len(folders), len(objects))
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

func TestRenameAndMoveObject(t *testing.T) {
	ctx := context.Background()
	store := drive.NewMemoryStore()
	owner := drive.Principal{TenantID: "t1", UserID: "u1"}
	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID: "t1", Name: "a.txt", OwnerUserID: "u1", Data: []byte("x"),
	}, "k1")
	if err != nil {
		t.Fatal(err)
	}
	folder, err := store.CreateFolder(ctx, "t1", "", "Archive", "u1")
	if err != nil {
		t.Fatal(err)
	}
	newName := "b.txt"
	folderID := folder.ID
	updated, err := store.UpdateObject(ctx, "t1", obj.ID, owner, drive.ObjectPatch{
		Name: &newName, FolderID: &folderID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "b.txt" || updated.FolderID != folder.ID {
		t.Fatalf("patch: %+v", updated)
	}
	renamed := "Archive2"
	f2, err := store.UpdateFolder(ctx, "t1", folder.ID, owner, drive.FolderPatch{Name: &renamed})
	if err != nil {
		t.Fatal(err)
	}
	if f2.Name != "Archive2" {
		t.Fatalf("folder rename: %+v", f2)
	}
}

func TestTrashHideAndRestore(t *testing.T) {
	ctx := context.Background()
	store := drive.NewMemoryStore()
	owner := drive.Principal{TenantID: "t1", UserID: "u-owner"}

	folder, err := store.CreateFolder(ctx, "t1", "", "Docs", owner.UserID)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID: "t1", FolderID: folder.ID, Name: "memo.txt",
		OwnerUserID: owner.UserID, Data: []byte("x"),
	}, "k1")
	if err != nil {
		t.Fatal(err)
	}

	folders, objects, err := store.ListChildren(ctx, "t1", folder.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 0 || len(objects) != 1 || objects[0].ID != obj.ID {
		t.Fatalf("before trash: folders=%d objects=%+v", len(folders), objects)
	}

	trashed, err := store.TrashObject(ctx, "t1", obj.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if trashed.TrashedAt == nil || trashed.TrashRestoreFolderID != folder.ID {
		t.Fatalf("trash fields: %+v", trashed)
	}

	_, objects, err = store.ListChildren(ctx, "t1", folder.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 0 {
		t.Fatalf("trashed object still in ListChildren: %+v", objects)
	}

	tf, to, err := store.ListTrash(ctx, "t1", owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(tf) != 0 || len(to) != 1 || to[0].ID != obj.ID {
		t.Fatalf("ListTrash: folders=%+v objects=%+v", tf, to)
	}

	restored, err := store.RestoreObject(ctx, "t1", obj.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if restored.TrashedAt != nil || restored.FolderID != folder.ID {
		t.Fatalf("restore: %+v", restored)
	}

	_, objects, err = store.ListChildren(ctx, "t1", folder.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 || objects[0].ID != obj.ID {
		t.Fatalf("after restore: %+v", objects)
	}

	if _, err := store.TrashFolder(ctx, "t1", folder.ID, owner); err != nil {
		t.Fatal(err)
	}
	folders, _, err = store.ListChildren(ctx, "t1", "", owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 0 {
		t.Fatalf("trashed folder still listed: %+v", folders)
	}
	if _, err := store.RestoreFolder(ctx, "t1", folder.ID, owner); err != nil {
		t.Fatal(err)
	}
}

func TestObjectLockUnlock(t *testing.T) {
	ctx := context.Background()
	store := drive.NewMemoryStore()
	owner := drive.Principal{TenantID: "t1", UserID: "u-owner"}
	writer := drive.Principal{TenantID: "t1", UserID: "u-writer"}
	obj, err := store.CreateObject(ctx, drive.CreateObjectInput{
		TenantID: "t1", Name: "locked.txt", OwnerUserID: owner.UserID, Data: []byte("x"),
	}, "k1")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateACL(ctx, "t1", obj.ID, owner, []drive.ACLEntry{
		{Principal: "user:u-owner", Role: drive.ACLRoleOwner},
		{Principal: "user:u-writer", Role: drive.ACLRoleWrite},
	}); err != nil {
		t.Fatal(err)
	}

	locked := true
	got, err := store.UpdateObject(ctx, "t1", obj.ID, writer, drive.ObjectPatch{Locked: &locked})
	if err != nil {
		t.Fatal(err)
	}
	if got.LockedBy != writer.UserID || got.LockedAt == nil {
		t.Fatalf("lock: %+v", got)
	}

	newName := "renamed.txt"
	if _, err := store.UpdateObject(ctx, "t1", obj.ID, owner, drive.ObjectPatch{Name: &newName}); err != drive.ErrLocked {
		t.Fatalf("owner rename while locked by other: %v", err)
	}
	if _, err := store.UpdateObject(ctx, "t1", obj.ID, writer, drive.ObjectPatch{Name: &newName}); err != nil {
		t.Fatalf("locker rename: %v", err)
	}

	unlocked := false
	got, err = store.UpdateObject(ctx, "t1", obj.ID, owner, drive.ObjectPatch{Locked: &unlocked})
	if err != nil {
		t.Fatalf("owner unlock: %v", err)
	}
	if got.LockedBy != "" || got.LockedAt != nil {
		t.Fatalf("unlock: %+v", got)
	}
}
