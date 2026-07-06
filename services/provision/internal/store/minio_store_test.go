package store

import (
	"context"
	"testing"
	"time"

	"era/services/provision/internal/minio"
)

func TestMinIOStoreListImages(t *testing.T) {
	mock := &minio.MockLister{
		Objects: []minio.ObjectInfo{
			{Key: "images/ubuntu-22.04.iso", Size: 1024, LastModified: time.Now()},
			{Key: "images/win2022.wim", Size: 2048, LastModified: time.Now()},
		},
	}
	objs, err := mock.ListObjects(context.Background(), "era-provision", "images/")
	if err != nil {
		t.Fatal(err)
	}
	if len(objs) != 2 {
		t.Fatalf("objects: %d", len(objs))
	}

	base := NewMemory()
	st := NewMinIOStore(base, mock, "era-provision", "images/")
	list := st.ListImages()
	if len(list) < 4 {
		t.Fatalf("expected static+minio images, got %d", len(list))
	}
}
