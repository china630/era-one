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
	Principal string
	Role      ACLRole
}

// Folder metadata row.
type Folder struct {
	ID        string
	TenantID  string
	ParentID  string
	Name      string
	CreatedAt time.Time
}

// Object metadata row.
type Object struct {
	ID            string
	TenantID      string
	FolderID      string
	Name          string
	SizeBytes     int64
	ContentType   string
	ContentFormat int32
	Version       int32
	BlobKey       string
	OwnerUserID   string
	ACL           []ACLEntry
	UpdatedAt     time.Time
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
	CreateObject(ctx context.Context, in CreateObjectInput, blobKey string) (Object, error)
	GetObject(ctx context.Context, tenantID, objectID string, p Principal) (Object, error)
	GetObjectData(ctx context.Context, tenantID, objectID string, p Principal) (Object, []byte, error)
	ListVersions(ctx context.Context, tenantID, objectID string, p Principal) ([]Version, error)
	UpdateACL(ctx context.Context, tenantID, objectID string, p Principal, entries []ACLEntry) error
	LookupObject(ctx context.Context, tenantID, objectID string) (Object, error)
}

// BlobStore stores opaque blob payloads.
type BlobStore interface {
	Put(data []byte) (key string, err error)
	Get(key string) ([]byte, error)
}
