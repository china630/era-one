package drive

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore is an in-process Store for unit tests and dev.
type MemoryStore struct {
	mu      sync.RWMutex
	folders map[string]Folder
	objects map[string]Object
	vers    map[string][]Version
	blobs   map[string][]byte
}

// NewMemoryStore creates an empty memory store with blob backing.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		folders: make(map[string]Folder),
		objects: make(map[string]Object),
		vers:    make(map[string][]Version),
		blobs:   make(map[string][]byte),
	}
}

func (m *MemoryStore) CreateFolder(_ context.Context, tenantID, parentID, name, ownerID string) (Folder, error) {
	if tenantID == "" || name == "" {
		return Folder{}, ErrInvalidInput
	}
	f := Folder{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		ParentID:  parentID,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.folders {
		if existing.TenantID == tenantID && existing.ParentID == parentID && existing.Name == name {
			return Folder{}, ErrDuplicate
		}
	}
	m.folders[f.ID] = f
	_ = ownerID
	return f, nil
}

func (m *MemoryStore) ListChildren(_ context.Context, tenantID, folderID string, p Principal) ([]Folder, []Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var folders []Folder
	var objects []Object
	for _, f := range m.folders {
		if f.TenantID == tenantID && f.ParentID == folderID && f.TrashedAt == nil {
			folders = append(folders, f)
		}
	}
	for _, o := range m.objects {
		if o.TenantID == tenantID && o.FolderID == folderID && o.TrashedAt == nil && CanRead(o, p) {
			objects = append(objects, o)
		}
	}
	return folders, objects, nil
}

func (m *MemoryStore) Search(_ context.Context, tenantID, query string, p Principal) ([]Folder, []Object, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil, nil
	}
	needle := strings.ToLower(q)
	m.mu.RLock()
	defer m.mu.RUnlock()
	var folders []Folder
	var objects []Object
	for _, f := range m.folders {
		if f.TenantID == tenantID && f.TrashedAt == nil && strings.Contains(strings.ToLower(f.Name), needle) {
			folders = append(folders, f)
		}
	}
	for _, o := range m.objects {
		if o.TenantID == tenantID && o.TrashedAt == nil && strings.Contains(strings.ToLower(o.Name), needle) && CanRead(o, p) {
			objects = append(objects, o)
		}
	}
	return folders, objects, nil
}

func (m *MemoryStore) CreateObject(_ context.Context, in CreateObjectInput, blobKey string) (Object, error) {
	if in.TenantID == "" || in.Name == "" || in.OwnerUserID == "" {
		return Object{}, ErrInvalidInput
	}
	if blobKey == "" {
		blobKey = "mem/" + uuid.NewString()
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
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.objects {
		if existing.TenantID == in.TenantID && existing.FolderID == in.FolderID && existing.Name == in.Name {
			return Object{}, ErrDuplicate
		}
	}
	m.objects[obj.ID] = obj
	m.blobs[blobKey] = append([]byte(nil), in.Data...)
	m.vers[obj.ID] = []Version{{
		Version:   1,
		BlobKey:   blobKey,
		SizeBytes: obj.SizeBytes,
		CreatedAt: now,
	}}
	return obj, nil
}

func (m *MemoryStore) PutVersion(_ context.Context, tenantID, objectID string, p Principal, in PutVersionInput, blobKey string) (Object, error) {
	if blobKey == "" {
		blobKey = "mem/" + uuid.NewString()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	if !CanWrite(obj, p) {
		return Object{}, ErrForbidden
	}
	if !CanMutateWhileLocked(obj, p) {
		return Object{}, ErrLocked
	}
	now := time.Now().UTC()
	obj.Version++
	obj.BlobKey = blobKey
	obj.SizeBytes = int64(len(in.Data))
	obj.UpdatedAt = now
	if strings.TrimSpace(in.ContentType) != "" {
		obj.ContentType = in.ContentType
	}
	m.objects[objectID] = obj
	m.blobs[blobKey] = append([]byte(nil), in.Data...)
	m.vers[objectID] = append(m.vers[objectID], Version{
		Version:   obj.Version,
		BlobKey:   blobKey,
		SizeBytes: obj.SizeBytes,
		CreatedAt: now,
	})
	return obj, nil
}

func (m *MemoryStore) GetObject(_ context.Context, tenantID, objectID string, p Principal) (Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, ok := m.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	if !CanRead(obj, p) {
		return Object{}, ErrForbidden
	}
	return obj, nil
}

func (m *MemoryStore) GetObjectData(ctx context.Context, tenantID, objectID string, p Principal) (Object, []byte, error) {
	obj, err := m.GetObject(ctx, tenantID, objectID, p)
	if err != nil {
		return Object{}, nil, err
	}
	m.mu.RLock()
	data := append([]byte(nil), m.blobs[obj.BlobKey]...)
	m.mu.RUnlock()
	return obj, data, nil
}

func (m *MemoryStore) ListVersions(_ context.Context, tenantID, objectID string, p Principal) ([]Version, error) {
	obj, err := m.GetObject(context.Background(), tenantID, objectID, p)
	if err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	vs := append([]Version(nil), m.vers[obj.ID]...)
	return vs, nil
}

func (m *MemoryStore) UpdateACL(_ context.Context, tenantID, objectID string, p Principal, entries []ACLEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return ErrNotFound
	}
	if !CanWrite(obj, p) {
		return ErrForbidden
	}
	obj.ACL = append([]ACLEntry(nil), entries...)
	m.objects[objectID] = obj
	return nil
}

func (m *MemoryStore) UpdateObject(ctx context.Context, tenantID, objectID string, p Principal, patch ObjectPatch) (Object, error) {
	if patch.Trashed != nil {
		if *patch.Trashed {
			return m.TrashObject(ctx, tenantID, objectID, p)
		}
		return m.RestoreObject(ctx, tenantID, objectID, p)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	if !CanWrite(obj, p) {
		return Object{}, ErrForbidden
	}

	mutatingMeta := patch.Name != nil || patch.FolderID != nil
	if mutatingMeta && !CanMutateWhileLocked(obj, p) {
		return Object{}, ErrLocked
	}

	if patch.Locked != nil {
		if *patch.Locked {
			if IsLocked(obj) && !IsLocker(obj, p) {
				return Object{}, ErrLocked
			}
			now := time.Now().UTC()
			obj.LockedBy = p.UserID
			obj.LockedAt = &now
		} else {
			if IsLocked(obj) && !CanUnlock(obj, p) {
				return Object{}, ErrForbidden
			}
			obj.LockedBy = ""
			obj.LockedAt = nil
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
		for _, existing := range m.objects {
			if existing.ID == objectID || existing.TrashedAt != nil {
				continue
			}
			if existing.TenantID == tenantID && existing.FolderID == folderID && existing.Name == name {
				return Object{}, ErrDuplicate
			}
		}
	}
	obj.Name = name
	obj.FolderID = folderID
	obj.UpdatedAt = time.Now().UTC()
	m.objects[objectID] = obj
	return obj, nil
}

func (m *MemoryStore) UpdateFolder(ctx context.Context, tenantID, folderID string, p Principal, patch FolderPatch) (Folder, error) {
	if patch.Trashed != nil {
		if *patch.Trashed {
			return m.TrashFolder(ctx, tenantID, folderID, p)
		}
		return m.RestoreFolder(ctx, tenantID, folderID, p)
	}
	_ = p
	if folderID == "" {
		return Folder{}, ErrInvalidInput
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.folders[folderID]
	if !ok || f.TenantID != tenantID {
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
		for _, existing := range m.folders {
			if existing.ID == folderID || existing.TrashedAt != nil {
				continue
			}
			if existing.TenantID == tenantID && existing.ParentID == parentID && existing.Name == name {
				return Folder{}, ErrDuplicate
			}
		}
	}
	f.Name = name
	f.ParentID = parentID
	m.folders[folderID] = f
	return f, nil
}

func (m *MemoryStore) LookupObject(_ context.Context, tenantID, objectID string) (Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, ok := m.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	return obj, nil
}

func (m *MemoryStore) ListTrash(_ context.Context, tenantID string, p Principal) ([]Folder, []Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var folders []Folder
	var objects []Object
	for _, f := range m.folders {
		if f.TenantID == tenantID && f.TrashedAt != nil {
			folders = append(folders, f)
		}
	}
	for _, o := range m.objects {
		if o.TenantID == tenantID && o.TrashedAt != nil && CanRead(o, p) {
			objects = append(objects, o)
		}
	}
	return folders, objects, nil
}

func (m *MemoryStore) TrashObject(_ context.Context, tenantID, objectID string, p Principal) (Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	if !CanWrite(obj, p) {
		return Object{}, ErrForbidden
	}
	if obj.TrashedAt != nil {
		return obj, nil
	}
	now := time.Now().UTC()
	obj.TrashRestoreFolderID = obj.FolderID
	obj.TrashedAt = &now
	obj.UpdatedAt = now
	m.objects[objectID] = obj
	return obj, nil
}

func (m *MemoryStore) RestoreObject(_ context.Context, tenantID, objectID string, p Principal) (Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[objectID]
	if !ok || obj.TenantID != tenantID {
		return Object{}, ErrNotFound
	}
	if !CanWrite(obj, p) {
		return Object{}, ErrForbidden
	}
	if obj.TrashedAt == nil {
		return obj, nil
	}
	folderID := obj.TrashRestoreFolderID
	for _, existing := range m.objects {
		if existing.ID == objectID || existing.TrashedAt != nil {
			continue
		}
		if existing.TenantID == tenantID && existing.FolderID == folderID && existing.Name == obj.Name {
			return Object{}, ErrDuplicate
		}
	}
	obj.FolderID = folderID
	obj.TrashRestoreFolderID = ""
	obj.TrashedAt = nil
	obj.UpdatedAt = time.Now().UTC()
	m.objects[objectID] = obj
	return obj, nil
}

func (m *MemoryStore) TrashFolder(_ context.Context, tenantID, folderID string, p Principal) (Folder, error) {
	_ = p
	if folderID == "" {
		return Folder{}, ErrInvalidInput
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.folders[folderID]
	if !ok || f.TenantID != tenantID {
		return Folder{}, ErrNotFound
	}
	if f.TrashedAt != nil {
		return f, nil
	}
	now := time.Now().UTC()
	f.TrashRestoreParentID = f.ParentID
	f.TrashedAt = &now
	m.folders[folderID] = f
	return f, nil
}

func (m *MemoryStore) RestoreFolder(_ context.Context, tenantID, folderID string, p Principal) (Folder, error) {
	_ = p
	if folderID == "" {
		return Folder{}, ErrInvalidInput
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.folders[folderID]
	if !ok || f.TenantID != tenantID {
		return Folder{}, ErrNotFound
	}
	if f.TrashedAt == nil {
		return f, nil
	}
	parentID := f.TrashRestoreParentID
	for _, existing := range m.folders {
		if existing.ID == folderID || existing.TrashedAt != nil {
			continue
		}
		if existing.TenantID == tenantID && existing.ParentID == parentID && existing.Name == f.Name {
			return Folder{}, ErrDuplicate
		}
	}
	f.ParentID = parentID
	f.TrashRestoreParentID = ""
	f.TrashedAt = nil
	m.folders[folderID] = f
	return f, nil
}

// MemoryBlobStore implements BlobStore in RAM.
type MemoryBlobStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemoryBlobStore creates an empty blob store.
func NewMemoryBlobStore() *MemoryBlobStore {
	return &MemoryBlobStore{data: make(map[string][]byte)}
}

func (b *MemoryBlobStore) Put(data []byte) (string, error) {
	key := "mem/" + uuid.NewString()
	b.mu.Lock()
	b.data[key] = append([]byte(nil), data...)
	b.mu.Unlock()
	return key, nil
}

func (b *MemoryBlobStore) Get(key string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	d, ok := b.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte(nil), d...), nil
}
