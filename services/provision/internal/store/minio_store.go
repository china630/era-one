package store

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"era/services/provision/internal/minio"
)

// MinIOStore — memory PXE + образы из MinIO list.
type MinIOStore struct {
	base   Repository
	lister minio.Lister
	bucket string
	prefix string
	mu     sync.Mutex
	images []*OSImage
}

// NewMinIOStore оборачивает base store и подтягивает образы из MinIO.
func NewMinIOStore(base Repository, lister minio.Lister, bucket, prefix string) *MinIOStore {
	return &MinIOStore{base: base, lister: lister, bucket: bucket, prefix: prefix}
}

func (s *MinIOStore) sync(ctx context.Context) {
	if s.lister == nil {
		return
	}
	objs, err := s.lister.ListObjects(ctx, s.bucket, s.prefix)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.images = imagesFromObjects(objs, s.bucket)
	s.mu.Unlock()
}

func imagesFromObjects(objects []minio.ObjectInfo, bucket string) []*OSImage {
	var out []*OSImage
	for i, obj := range objects {
		name := obj.Key
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		platform := "linux"
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".wim") || (strings.HasSuffix(lower, ".iso") && strings.Contains(lower, "win")) {
			platform = "windows"
		}
		out = append(out, &OSImage{
			ID:        fmt.Sprintf("minio-%d", i),
			Name:      name,
			Platform:  platform,
			Version:   "unknown",
			MinIORef:  fmt.Sprintf("s3://%s/%s", bucket, obj.Key),
			CreatedAt: obj.LastModified.UTC(),
		})
	}
	return out
}

func (s *MinIOStore) ListImages() []*OSImage {
	s.sync(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	merged := append(s.base.ListImages(), s.images...)
	out := make([]*OSImage, len(merged))
	copy(out, merged)
	return out
}

func (s *MinIOStore) GetImage(id string) (*OSImage, bool) {
	s.sync(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, img := range s.images {
		if img.ID == id {
			return img, true
		}
	}
	return s.base.GetImage(id)
}

func (s *MinIOStore) PXEConfig() PXEConfig {
	return s.base.PXEConfig()
}
