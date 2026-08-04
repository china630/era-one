package drive

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PgStore persists drive metadata in era_platform schema.
type PgStore struct {
	db *sql.DB
}

// OpenPostgres connects to Postgres and verifies connectivity.
func OpenPostgres(dsn string) (*PgStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &PgStore{db: db}, nil
}

// Close releases the connection pool.
func (p *PgStore) Close() error {
	if p == nil || p.db == nil {
		return nil
	}
	return p.db.Close()
}

// OpenFromEnv returns Postgres store when ERA_OFFICE_DATABASE_URL is set, else memory.
func OpenFromEnv() (Store, func() error, error) {
	dsn := strings.TrimSpace(os.Getenv("ERA_OFFICE_DATABASE_URL"))
	if dsn == "" {
		return NewMemoryStore(), func() error { return nil }, nil
	}
	s, err := OpenPostgres(dsn)
	if err != nil {
		return nil, nil, err
	}
	return s, s.Close, nil
}

func (p *PgStore) ensureTenant(ctx context.Context, tenantID string) error {
	if tenantID == "" {
		tenantID = "t-demo"
	}
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO era_platform.tenants (id, name, slug, status) VALUES ($1,$2,$3,'active')
		 ON CONFLICT (id) DO NOTHING`,
		tenantID, tenantID, strings.ToLower(tenantID),
	)
	return err
}

func (p *PgStore) CreateFolder(ctx context.Context, tenantID, parentID, name, ownerID string) (Folder, error) {
	if tenantID == "" || name == "" {
		return Folder{}, ErrInvalidInput
	}
	if err := p.ensureTenant(ctx, tenantID); err != nil {
		return Folder{}, err
	}
	var exists int
	err := p.db.QueryRowContext(ctx,
		`SELECT 1 FROM era_platform.drive_folders WHERE tenant_id=$1 AND parent_id=$2 AND name=$3 LIMIT 1`,
		tenantID, parentID, name,
	).Scan(&exists)
	if err == nil {
		return Folder{}, ErrDuplicate
	}
	if err != sql.ErrNoRows {
		return Folder{}, err
	}
	f := Folder{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		ParentID:  parentID,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	_, err = p.db.ExecContext(ctx,
		`INSERT INTO era_platform.drive_folders (id, tenant_id, parent_id, name, created_at) VALUES ($1,$2,$3,$4,$5)`,
		f.ID, f.TenantID, f.ParentID, f.Name, f.CreatedAt,
	)
	if err != nil {
		return Folder{}, err
	}
	_ = ownerID
	return f, nil
}

func (p *PgStore) ListChildren(ctx context.Context, tenantID, folderID string, pr Principal) ([]Folder, []Object, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, tenant_id, parent_id, name, created_at, trashed_at, trash_restore_parent_id
		 FROM era_platform.drive_folders
		 WHERE tenant_id=$1 AND parent_id=$2 AND trashed_at IS NULL ORDER BY name`,
		tenantID, folderID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var folders []Folder
	for rows.Next() {
		f, err := scanFolderProps(rows)
		if err != nil {
			return nil, nil, err
		}
		folders = append(folders, f)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	objs, err := p.listObjectsInFolder(ctx, tenantID, folderID)
	if err != nil {
		return nil, nil, err
	}
	var visible []Object
	for _, o := range objs {
		if CanRead(o, pr) {
			visible = append(visible, o)
		}
	}
	return folders, visible, nil
}

func (p *PgStore) Search(ctx context.Context, tenantID, query string, pr Principal) ([]Folder, []Object, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil, nil
	}
	pattern := likeContainsPattern(q)

	rows, err := p.db.QueryContext(ctx,
		`SELECT id, tenant_id, parent_id, name, created_at, trashed_at, trash_restore_parent_id
		 FROM era_platform.drive_folders
		 WHERE tenant_id=$1 AND trashed_at IS NULL AND name ILIKE $2 ESCAPE '\' ORDER BY name`,
		tenantID, pattern,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var folders []Folder
	for rows.Next() {
		f, err := scanFolderProps(rows)
		if err != nil {
			return nil, nil, err
		}
		folders = append(folders, f)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	objRows, err := p.db.QueryContext(ctx,
		`SELECT id, tenant_id, folder_id, name, size_bytes, content_type, content_format,
		        version, blob_key, owner_user_id, updated_at, locked_by, locked_at,
		        trashed_at, trash_restore_folder_id
		 FROM era_platform.drive_objects
		 WHERE tenant_id=$1 AND trashed_at IS NULL AND name ILIKE $2 ESCAPE '\' ORDER BY name`,
		tenantID, pattern,
	)
	if err != nil {
		return nil, nil, err
	}
	defer objRows.Close()
	var visible []Object
	for objRows.Next() {
		obj, err := scanObjectProps(objRows)
		if err != nil {
			return nil, nil, err
		}
		acl, err := p.loadACL(ctx, obj.ID)
		if err != nil {
			return nil, nil, err
		}
		obj.ACL = acl
		if CanRead(obj, pr) {
			visible = append(visible, obj)
		}
	}
	if err := objRows.Err(); err != nil {
		return nil, nil, err
	}
	return folders, visible, nil
}

// likeContainsPattern builds an ILIKE pattern for case-insensitive substring match.
func likeContainsPattern(q string) string {
	q = strings.ReplaceAll(q, `\`, `\\`)
	q = strings.ReplaceAll(q, `%`, `\%`)
	q = strings.ReplaceAll(q, `_`, `\_`)
	return "%" + q + "%"
}

func (p *PgStore) listObjectsInFolder(ctx context.Context, tenantID, folderID string) ([]Object, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, tenant_id, folder_id, name, size_bytes, content_type, content_format,
		        version, blob_key, owner_user_id, updated_at, locked_by, locked_at,
		        trashed_at, trash_restore_folder_id
		 FROM era_platform.drive_objects
		 WHERE tenant_id=$1 AND folder_id=$2 AND trashed_at IS NULL ORDER BY name`,
		tenantID, folderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Object
	for rows.Next() {
		obj, err := scanObjectProps(rows)
		if err != nil {
			return nil, err
		}
		acl, err := p.loadACL(ctx, obj.ID)
		if err != nil {
			return nil, err
		}
		obj.ACL = acl
		out = append(out, obj)
	}
	return out, rows.Err()
}

func (p *PgStore) CreateObject(ctx context.Context, in CreateObjectInput, blobKey string) (Object, error) {
	if in.TenantID == "" || in.Name == "" || in.OwnerUserID == "" {
		return Object{}, ErrInvalidInput
	}
	if blobKey == "" {
		blobKey = "pg/" + uuid.NewString()
	}
	if err := p.ensureTenant(ctx, in.TenantID); err != nil {
		return Object{}, err
	}
	var exists int
	err := p.db.QueryRowContext(ctx,
		`SELECT 1 FROM era_platform.drive_objects WHERE tenant_id=$1 AND folder_id=$2 AND name=$3 LIMIT 1`,
		in.TenantID, in.FolderID, in.Name,
	).Scan(&exists)
	if err == nil {
		return Object{}, ErrDuplicate
	}
	if err != sql.ErrNoRows {
		return Object{}, err
	}

	now := time.Now().UTC()
	obj := Object{
		ID:            uuid.NewString(),
		TenantID:      in.TenantID,
		FolderID:      in.FolderID,
		Name:          in.Name,
		SizeBytes:     int64(len(in.Data)),
		ContentType:   in.ContentType,
		ContentFormat: in.ContentFormat,
		Version:       1,
		BlobKey:       blobKey,
		OwnerUserID:   in.OwnerUserID,
		ACL:           DefaultOwnerACL(in.TenantID, in.OwnerUserID),
		UpdatedAt:     now,
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return Object{}, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO era_platform.drive_objects
		 (id, tenant_id, folder_id, name, size_bytes, content_type, content_format, version, blob_key, owner_user_id, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		obj.ID, obj.TenantID, obj.FolderID, obj.Name, obj.SizeBytes, obj.ContentType, obj.ContentFormat,
		obj.Version, obj.BlobKey, obj.OwnerUserID, obj.UpdatedAt,
	)
	if err != nil {
		return Object{}, err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO era_platform.drive_versions (object_id, version, blob_key, size_bytes, created_at) VALUES ($1,$2,$3,$4,$5)`,
		obj.ID, obj.Version, obj.BlobKey, obj.SizeBytes, now,
	)
	if err != nil {
		return Object{}, err
	}
	for _, e := range obj.ACL {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO era_platform.drive_acl (object_id, principal, role) VALUES ($1,$2,$3)`,
			obj.ID, e.Principal, int32(e.Role),
		); err != nil {
			return Object{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Object{}, err
	}
	return obj, nil
}

func (p *PgStore) PutVersion(ctx context.Context, tenantID, objectID string, pr Principal, in PutVersionInput, blobKey string) (Object, error) {
	if blobKey == "" {
		return Object{}, ErrInvalidInput
	}
	obj, err := p.fetchObject(ctx, objectID)
	if err != nil {
		return Object{}, err
	}
	if obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	if !CanWrite(obj, pr) {
		return Object{}, ErrForbidden
	}
	if !CanMutateWhileLocked(obj, pr) {
		return Object{}, ErrLocked
	}
	now := time.Now().UTC()
	nextVer := obj.Version + 1
	sizeBytes := int64(len(in.Data))
	contentType := obj.ContentType
	if strings.TrimSpace(in.ContentType) != "" {
		contentType = in.ContentType
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return Object{}, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`UPDATE era_platform.drive_objects
		 SET version=$1, blob_key=$2, size_bytes=$3, content_type=$4, updated_at=$5
		 WHERE id=$6 AND tenant_id=$7`,
		nextVer, blobKey, sizeBytes, contentType, now, objectID, tenantID,
	)
	if err != nil {
		return Object{}, err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO era_platform.drive_versions (object_id, version, blob_key, size_bytes, created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		objectID, nextVer, blobKey, sizeBytes, now,
	)
	if err != nil {
		return Object{}, err
	}
	if err := tx.Commit(); err != nil {
		return Object{}, err
	}
	obj.Version = nextVer
	obj.BlobKey = blobKey
	obj.SizeBytes = sizeBytes
	obj.ContentType = contentType
	obj.UpdatedAt = now
	return obj, nil
}

func (p *PgStore) GetObject(ctx context.Context, tenantID, objectID string, pr Principal) (Object, error) {
	obj, err := p.fetchObject(ctx, objectID)
	if err != nil {
		return Object{}, err
	}
	if obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	if !CanRead(obj, pr) {
		return Object{}, ErrForbidden
	}
	return obj, nil
}

func (p *PgStore) LookupObject(ctx context.Context, tenantID, objectID string) (Object, error) {
	obj, err := p.fetchObject(ctx, objectID)
	if err != nil {
		return Object{}, err
	}
	if obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	return obj, nil
}

func (p *PgStore) GetObjectData(ctx context.Context, tenantID, objectID string, pr Principal) (Object, []byte, error) {
	obj, err := p.GetObject(ctx, tenantID, objectID, pr)
	if err != nil {
		return Object{}, nil, err
	}
	return obj, nil, nil
}

func (p *PgStore) ListVersions(ctx context.Context, tenantID, objectID string, pr Principal) ([]Version, error) {
	if _, err := p.GetObject(ctx, tenantID, objectID, pr); err != nil {
		return nil, err
	}
	rows, err := p.db.QueryContext(ctx,
		`SELECT version, blob_key, size_bytes, created_at FROM era_platform.drive_versions
		 WHERE object_id=$1 ORDER BY version`,
		objectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vs []Version
	for rows.Next() {
		var v Version
		if err := rows.Scan(&v.Version, &v.BlobKey, &v.SizeBytes, &v.CreatedAt); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, rows.Err()
}

func (p *PgStore) UpdateACL(ctx context.Context, tenantID, objectID string, pr Principal, entries []ACLEntry) error {
	obj, err := p.fetchObject(ctx, objectID)
	if err != nil {
		return err
	}
	if obj.TenantID != tenantID {
		return ErrNotFound
	}
	if !CanWrite(obj, pr) {
		return ErrForbidden
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM era_platform.drive_acl WHERE object_id=$1`, objectID); err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO era_platform.drive_acl (object_id, principal, role) VALUES ($1,$2,$3)`,
			objectID, e.Principal, int32(e.Role),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *PgStore) UpdateObject(ctx context.Context, tenantID, objectID string, pr Principal, patch ObjectPatch) (Object, error) {
	if patch.Trashed != nil {
		if *patch.Trashed {
			return p.TrashObject(ctx, tenantID, objectID, pr)
		}
		return p.RestoreObject(ctx, tenantID, objectID, pr)
	}
	obj, err := p.GetObject(ctx, tenantID, objectID, pr)
	if err != nil {
		return Object{}, err
	}
	if !CanWrite(obj, pr) {
		return Object{}, ErrForbidden
	}

	mutatingMeta := patch.Name != nil || patch.FolderID != nil
	if mutatingMeta && !CanMutateWhileLocked(obj, pr) {
		return Object{}, ErrLocked
	}

	lockedBy := obj.LockedBy
	var lockedAt *time.Time
	if obj.LockedAt != nil {
		t := *obj.LockedAt
		lockedAt = &t
	}
	if patch.Locked != nil {
		if *patch.Locked {
			if IsLocked(obj) && !IsLocker(obj, pr) {
				return Object{}, ErrLocked
			}
			nowLock := time.Now().UTC()
			lockedBy = pr.UserID
			lockedAt = &nowLock
		} else {
			if IsLocked(obj) && !CanUnlock(obj, pr) {
				return Object{}, ErrForbidden
			}
			lockedBy = ""
			lockedAt = nil
		}
	}

	name := obj.Name
	folderID := obj.FolderID
	if patch.Name != nil {
		name = strings.TrimSpace(*patch.Name)
		if name == "" {
			return Object{}, ErrInvalidInput
		}
	}
	if patch.FolderID != nil {
		folderID = *patch.FolderID
	}
	if name != obj.Name || folderID != obj.FolderID {
		var exists int
		err := p.db.QueryRowContext(ctx,
			`SELECT 1 FROM era_platform.drive_objects WHERE tenant_id=$1 AND folder_id=$2 AND name=$3 AND id<>$4 LIMIT 1`,
			tenantID, folderID, name, objectID,
		).Scan(&exists)
		if err == nil {
			return Object{}, ErrDuplicate
		}
		if err != sql.ErrNoRows {
			return Object{}, err
		}
	}
	now := time.Now().UTC()
	var lockedByArg any
	var lockedAtArg any
	if lockedBy != "" {
		lockedByArg = lockedBy
		if lockedAt != nil {
			lockedAtArg = *lockedAt
		} else {
			lockedAtArg = now
		}
	} else {
		lockedByArg = nil
		lockedAtArg = nil
	}
	_, err = p.db.ExecContext(ctx,
		`UPDATE era_platform.drive_objects
		 SET name=$1, folder_id=$2, updated_at=$3, locked_by=$4, locked_at=$5
		 WHERE id=$6 AND tenant_id=$7`,
		name, folderID, now, lockedByArg, lockedAtArg, objectID, tenantID,
	)
	if err != nil {
		return Object{}, err
	}
	obj.Name = name
	obj.FolderID = folderID
	obj.UpdatedAt = now
	obj.LockedBy = lockedBy
	obj.LockedAt = lockedAt
	return obj, nil
}

func (p *PgStore) UpdateFolder(ctx context.Context, tenantID, folderID string, pr Principal, patch FolderPatch) (Folder, error) {
	if patch.Trashed != nil {
		if *patch.Trashed {
			return p.TrashFolder(ctx, tenantID, folderID, pr)
		}
		return p.RestoreFolder(ctx, tenantID, folderID, pr)
	}
	_ = pr
	if folderID == "" {
		return Folder{}, ErrInvalidInput
	}
	f, err := p.fetchFolder(ctx, folderID)
	if err != nil {
		return Folder{}, err
	}
	if f.TenantID != tenantID {
		return Folder{}, ErrNotFound
	}
	name := f.Name
	parentID := f.ParentID
	if patch.Name != nil {
		name = strings.TrimSpace(*patch.Name)
		if name == "" {
			return Folder{}, ErrInvalidInput
		}
	}
	if patch.ParentID != nil {
		parentID = *patch.ParentID
		if parentID == folderID {
			return Folder{}, ErrInvalidInput
		}
	}
	if name != f.Name || parentID != f.ParentID {
		var exists int
		err := p.db.QueryRowContext(ctx,
			`SELECT 1 FROM era_platform.drive_folders
			 WHERE tenant_id=$1 AND parent_id=$2 AND name=$3 AND id<>$4 AND trashed_at IS NULL LIMIT 1`,
			tenantID, parentID, name, folderID,
		).Scan(&exists)
		if err == nil {
			return Folder{}, ErrDuplicate
		}
		if err != sql.ErrNoRows {
			return Folder{}, err
		}
	}
	_, err = p.db.ExecContext(ctx,
		`UPDATE era_platform.drive_folders SET name=$1, parent_id=$2 WHERE id=$3 AND tenant_id=$4`,
		name, parentID, folderID, tenantID,
	)
	if err != nil {
		return Folder{}, err
	}
	f.Name = name
	f.ParentID = parentID
	return f, nil
}

func (p *PgStore) ListTrash(ctx context.Context, tenantID string, pr Principal) ([]Folder, []Object, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, tenant_id, parent_id, name, created_at, trashed_at, trash_restore_parent_id
		 FROM era_platform.drive_folders
		 WHERE tenant_id=$1 AND trashed_at IS NOT NULL ORDER BY trashed_at DESC, name`,
		tenantID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var folders []Folder
	for rows.Next() {
		f, err := scanFolderProps(rows)
		if err != nil {
			return nil, nil, err
		}
		folders = append(folders, f)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	objRows, err := p.db.QueryContext(ctx,
		`SELECT id, tenant_id, folder_id, name, size_bytes, content_type, content_format,
		        version, blob_key, owner_user_id, updated_at, locked_by, locked_at,
		        trashed_at, trash_restore_folder_id
		 FROM era_platform.drive_objects
		 WHERE tenant_id=$1 AND trashed_at IS NOT NULL ORDER BY trashed_at DESC, name`,
		tenantID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer objRows.Close()
	var visible []Object
	for objRows.Next() {
		obj, err := scanObjectProps(objRows)
		if err != nil {
			return nil, nil, err
		}
		acl, err := p.loadACL(ctx, obj.ID)
		if err != nil {
			return nil, nil, err
		}
		obj.ACL = acl
		if CanRead(obj, pr) {
			visible = append(visible, obj)
		}
	}
	if err := objRows.Err(); err != nil {
		return nil, nil, err
	}
	return folders, visible, nil
}

func (p *PgStore) TrashObject(ctx context.Context, tenantID, objectID string, pr Principal) (Object, error) {
	obj, err := p.fetchObject(ctx, objectID)
	if err != nil {
		return Object{}, err
	}
	if obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	if !CanWrite(obj, pr) {
		return Object{}, ErrForbidden
	}
	if obj.TrashedAt != nil {
		return obj, nil
	}
	now := time.Now().UTC()
	_, err = p.db.ExecContext(ctx,
		`UPDATE era_platform.drive_objects
		 SET trashed_at=$1, trash_restore_folder_id=$2, updated_at=$1
		 WHERE id=$3 AND tenant_id=$4`,
		now, obj.FolderID, objectID, tenantID,
	)
	if err != nil {
		return Object{}, err
	}
	obj.TrashRestoreFolderID = obj.FolderID
	obj.TrashedAt = &now
	obj.UpdatedAt = now
	return obj, nil
}

func (p *PgStore) RestoreObject(ctx context.Context, tenantID, objectID string, pr Principal) (Object, error) {
	obj, err := p.fetchObject(ctx, objectID)
	if err != nil {
		return Object{}, err
	}
	if obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	if !CanWrite(obj, pr) {
		return Object{}, ErrForbidden
	}
	if obj.TrashedAt == nil {
		return obj, nil
	}
	folderID := obj.TrashRestoreFolderID
	var exists int
	err = p.db.QueryRowContext(ctx,
		`SELECT 1 FROM era_platform.drive_objects
		 WHERE tenant_id=$1 AND folder_id=$2 AND name=$3 AND id<>$4 AND trashed_at IS NULL LIMIT 1`,
		tenantID, folderID, obj.Name, objectID,
	).Scan(&exists)
	if err == nil {
		return Object{}, ErrDuplicate
	}
	if err != sql.ErrNoRows {
		return Object{}, err
	}
	now := time.Now().UTC()
	_, err = p.db.ExecContext(ctx,
		`UPDATE era_platform.drive_objects
		 SET folder_id=$1, trashed_at=NULL, trash_restore_folder_id=NULL, updated_at=$2
		 WHERE id=$3 AND tenant_id=$4`,
		folderID, now, objectID, tenantID,
	)
	if err != nil {
		return Object{}, err
	}
	obj.FolderID = folderID
	obj.TrashRestoreFolderID = ""
	obj.TrashedAt = nil
	obj.UpdatedAt = now
	return obj, nil
}

func (p *PgStore) TrashFolder(ctx context.Context, tenantID, folderID string, pr Principal) (Folder, error) {
	_ = pr
	if folderID == "" {
		return Folder{}, ErrInvalidInput
	}
	f, err := p.fetchFolder(ctx, folderID)
	if err != nil {
		return Folder{}, err
	}
	if f.TenantID != tenantID {
		return Folder{}, ErrNotFound
	}
	if f.TrashedAt != nil {
		return f, nil
	}
	now := time.Now().UTC()
	_, err = p.db.ExecContext(ctx,
		`UPDATE era_platform.drive_folders
		 SET trashed_at=$1, trash_restore_parent_id=$2
		 WHERE id=$3 AND tenant_id=$4`,
		now, f.ParentID, folderID, tenantID,
	)
	if err != nil {
		return Folder{}, err
	}
	f.TrashRestoreParentID = f.ParentID
	f.TrashedAt = &now
	return f, nil
}

func (p *PgStore) RestoreFolder(ctx context.Context, tenantID, folderID string, pr Principal) (Folder, error) {
	_ = pr
	if folderID == "" {
		return Folder{}, ErrInvalidInput
	}
	f, err := p.fetchFolder(ctx, folderID)
	if err != nil {
		return Folder{}, err
	}
	if f.TenantID != tenantID {
		return Folder{}, ErrNotFound
	}
	if f.TrashedAt == nil {
		return f, nil
	}
	parentID := f.TrashRestoreParentID
	var exists int
	err = p.db.QueryRowContext(ctx,
		`SELECT 1 FROM era_platform.drive_folders
		 WHERE tenant_id=$1 AND parent_id=$2 AND name=$3 AND id<>$4 AND trashed_at IS NULL LIMIT 1`,
		tenantID, parentID, f.Name, folderID,
	).Scan(&exists)
	if err == nil {
		return Folder{}, ErrDuplicate
	}
	if err != sql.ErrNoRows {
		return Folder{}, err
	}
	_, err = p.db.ExecContext(ctx,
		`UPDATE era_platform.drive_folders
		 SET parent_id=$1, trashed_at=NULL, trash_restore_parent_id=NULL
		 WHERE id=$2 AND tenant_id=$3`,
		parentID, folderID, tenantID,
	)
	if err != nil {
		return Folder{}, err
	}
	f.ParentID = parentID
	f.TrashRestoreParentID = ""
	f.TrashedAt = nil
	return f, nil
}

func (p *PgStore) fetchFolder(ctx context.Context, folderID string) (Folder, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, parent_id, name, created_at, trashed_at, trash_restore_parent_id
		 FROM era_platform.drive_folders WHERE id=$1`,
		folderID,
	)
	f, err := scanFolderProps(row)
	if err == sql.ErrNoRows {
		return Folder{}, ErrNotFound
	}
	if err != nil {
		return Folder{}, err
	}
	return f, nil
}

func (p *PgStore) fetchObject(ctx context.Context, objectID string) (Object, error) {
	row := p.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, folder_id, name, size_bytes, content_type, content_format,
		        version, blob_key, owner_user_id, updated_at, locked_by, locked_at,
		        trashed_at, trash_restore_folder_id
		 FROM era_platform.drive_objects WHERE id=$1`,
		objectID,
	)
	obj, err := scanObjectProps(row)
	if err == sql.ErrNoRows {
		return Object{}, ErrNotFound
	}
	if err != nil {
		return Object{}, err
	}
	obj.ACL, err = p.loadACL(ctx, objectID)
	if err != nil {
		return Object{}, err
	}
	return obj, nil
}

func (p *PgStore) loadACL(ctx context.Context, objectID string) ([]ACLEntry, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT principal, role FROM era_platform.drive_acl WHERE object_id=$1 ORDER BY principal`,
		objectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ACLEntry
	for rows.Next() {
		var e ACLEntry
		var role int32
		if err := rows.Scan(&e.Principal, &role); err != nil {
			return nil, err
		}
		e.Role = ACLRole(role)
		out = append(out, e)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFolderProps(row rowScanner) (Folder, error) {
	var f Folder
	var trashedAt sql.NullTime
	var restoreParent sql.NullString
	if err := row.Scan(
		&f.ID, &f.TenantID, &f.ParentID, &f.Name, &f.CreatedAt,
		&trashedAt, &restoreParent,
	); err != nil {
		return Folder{}, err
	}
	if trashedAt.Valid {
		t := trashedAt.Time.UTC()
		f.TrashedAt = &t
	}
	if restoreParent.Valid {
		f.TrashRestoreParentID = restoreParent.String
	}
	return f, nil
}

func scanObjectProps(row rowScanner) (Object, error) {
	var obj Object
	var contentFormat int
	var lockedBy sql.NullString
	var lockedAt sql.NullTime
	var trashedAt sql.NullTime
	var restoreFolder sql.NullString
	if err := row.Scan(
		&obj.ID, &obj.TenantID, &obj.FolderID, &obj.Name, &obj.SizeBytes,
		&obj.ContentType, &contentFormat, &obj.Version, &obj.BlobKey, &obj.OwnerUserID, &obj.UpdatedAt,
		&lockedBy, &lockedAt, &trashedAt, &restoreFolder,
	); err != nil {
		return Object{}, err
	}
	obj.ContentFormat = int32(contentFormat)
	if lockedBy.Valid {
		obj.LockedBy = lockedBy.String
	}
	if lockedAt.Valid {
		t := lockedAt.Time.UTC()
		obj.LockedAt = &t
	}
	if trashedAt.Valid {
		t := trashedAt.Time.UTC()
		obj.TrashedAt = &t
	}
	if restoreFolder.Valid {
		obj.TrashRestoreFolderID = restoreFolder.String
	}
	return obj, nil
}

// EnsureSchema verifies era_platform.drive_objects exists (for health checks).
func (p *PgStore) EnsureSchema(ctx context.Context) error {
	var n int
	err := p.db.QueryRowContext(ctx,
		`SELECT 1 FROM information_schema.tables WHERE table_schema='era_platform' AND table_name='drive_objects' LIMIT 1`,
	).Scan(&n)
	if err == sql.ErrNoRows {
		return fmt.Errorf("drive: era_platform schema not migrated")
	}
	return err
}
