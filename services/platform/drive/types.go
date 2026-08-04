// Package drive — ERA Drive metadata + blob orchestration (Office P0).
package drive

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("drive: not found")
	ErrForbidden     = errors.New("drive: forbidden")
	ErrDuplicate     = errors.New("drive: duplicate name")
	ErrInvalidInput  = errors.New("drive: invalid input")
	ErrLicenseDenied = errors.New("drive: license required")
	ErrLocked        = errors.New("drive: object locked")
)

// ACLRole mirrors proto DriveACLRole.
type ACLRole int32

const (
	ACLRoleOwner ACLRole = 1
	ACLRoleRead  ACLRole = 2
	ACLRoleWrite ACLRole = 3
)

// ACLEntry is one ACL principal binding.
type ACLEntry struct {
	Principal string  `json:"principal"`
	Role      ACLRole `json:"role"`
}

// ObjectPatch updates object name and/or folder (nil = leave unchanged).
// Locked: true = lock as current user; false = unlock; nil = leave unchanged.
// Trashed: true = soft-delete; false = restore; nil = leave unchanged.
type ObjectPatch struct {
	Name     *string `json:"name"`
	FolderID *string `json:"folder_id"`
	Locked   *bool   `json:"locked"`
	Trashed  *bool   `json:"trashed"`
}

// FolderPatch updates folder name and/or parent (nil = leave unchanged).
// Trashed: true = soft-delete; false = restore; nil = leave unchanged.
type FolderPatch struct {
	Name     *string `json:"name"`
	ParentID *string `json:"parent_id"`
	Trashed  *bool   `json:"trashed"`
}

// Folder metadata row.
type Folder struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id,omitempty"`
	ParentID  string     `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
	TrashedAt *time.Time `json:"trashed_at,omitempty"`
	// TrashRestoreParentID is the parent before trash; empty = root.
	TrashRestoreParentID string `json:"trash_restore_parent_id,omitempty"`
}

// Object metadata row.
type Object struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id,omitempty"`
	FolderID      string     `json:"folder_id,omitempty"`
	Name          string     `json:"name"`
	SizeBytes     int64      `json:"size_bytes"`
	ContentType   string     `json:"content_type,omitempty"`
	ContentFormat int32      `json:"content_format,omitempty"`
	Version       int32      `json:"version"`
	BlobKey       string     `json:"blob_key,omitempty"`
	OwnerUserID   string     `json:"owner_user_id,omitempty"`
	ACL           []ACLEntry `json:"acl,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at,omitempty"`
	// LockedBy is the user id holding the lock; empty = unlocked.
	LockedBy string     `json:"locked_by,omitempty"`
	LockedAt *time.Time `json:"locked_at,omitempty"`
	TrashedAt *time.Time `json:"trashed_at,omitempty"`
	// TrashRestoreFolderID is the folder before trash; empty = root.
	TrashRestoreFolderID string `json:"trash_restore_folder_id,omitempty"`
}

// Version is one immutable blob revision.
type Version struct {
	Version   int32
	BlobKey   string
	SizeBytes int64
	CreatedAt time.Time
}

// CreateObjectInput for uploads.
type CreateObjectInput struct {
	TenantID      string
	FolderID      string
	Name          string
	ContentType   string
	ContentFormat int32
	OwnerUserID   string
	Data          []byte
}

// PutVersionInput for content re-upload (new version, stable object id).
type PutVersionInput struct {
	// Data is stored inline by MemoryStore; PgStore relies on BlobStore via blobKey.
	Data []byte
	// ContentType if non-empty updates the object content type.
	ContentType string
}

// Principal carries authenticated caller identity for ACL checks.
type Principal struct {
	TenantID string
	UserID   string
	Groups   []string
}

// Store persists drive metadata.
type Store interface {
	CreateFolder(ctx context.Context, tenantID, parentID, name, ownerID string) (Folder, error)
	ListChildren(ctx context.Context, tenantID, folderID string, p Principal) ([]Folder, []Object, error)
	// Search finds folders/objects by case-insensitive name substring within the tenant.
	// Objects are filtered by ACL (CanRead); folders are tenant-scoped like ListChildren.
	Search(ctx context.Context, tenantID, query string, p Principal) ([]Folder, []Object, error)
	CreateObject(ctx context.Context, in CreateObjectInput, blobKey string) (Object, error)
	// PutVersion appends an immutable content revision and advances the object head
	// (same object id / name). Requires CanWrite + CanMutateWhileLocked.
	PutVersion(ctx context.Context, tenantID, objectID string, p Principal, in PutVersionInput, blobKey string) (Object, error)
	GetObject(ctx context.Context, tenantID, objectID string, p Principal) (Object, error)
	GetObjectData(ctx context.Context, tenantID, objectID string, p Principal) (Object, []byte, error)
	ListVersions(ctx context.Context, tenantID, objectID string, p Principal) ([]Version, error)
	UpdateACL(ctx context.Context, tenantID, objectID string, p Principal, entries []ACLEntry) error
	UpdateObject(ctx context.Context, tenantID, objectID string, p Principal, patch ObjectPatch) (Object, error)
	UpdateFolder(ctx context.Context, tenantID, folderID string, p Principal, patch FolderPatch) (Folder, error)
	LookupObject(ctx context.Context, tenantID, objectID string) (Object, error)
	// ListTrash returns soft-deleted folders and objects for the tenant.
	// Objects are filtered by ACL (CanRead); folders are tenant-scoped.
	ListTrash(ctx context.Context, tenantID string, p Principal) ([]Folder, []Object, error)
	TrashObject(ctx context.Context, tenantID, objectID string, p Principal) (Object, error)
	RestoreObject(ctx context.Context, tenantID, objectID string, p Principal) (Object, error)
	TrashFolder(ctx context.Context, tenantID, folderID string, p Principal) (Folder, error)
	RestoreFolder(ctx context.Context, tenantID, folderID string, p Principal) (Folder, error)
}

// BlobStore stores opaque blob payloads.
type BlobStore interface {
	Put(data []byte) (key string, err error)
	Get(key string) ([]byte, error)
}
