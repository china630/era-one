// Package blobstore — MinIO/S3 blob storage for ERA Drive (Office P0).
package blobstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Store uploads and downloads drive blobs.
type Store interface {
	Put(data []byte) (key string, err error)
	Get(key string) ([]byte, error)
	Ping() error
}

// MinIO implements Store against S3 API.
type MinIO struct {
	client *minio.Client
	bucket string
}

// OpenFromEnv returns MinIO store or nil if ERA_MINIO_ENDPOINT unset.
func OpenFromEnv() (Store, error) {
	endpoint := strings.TrimSpace(os.Getenv("ERA_MINIO_ENDPOINT"))
	if endpoint == "" {
		return nil, nil
	}
	access := os.Getenv("ERA_MINIO_ACCESS_KEY")
	secret := os.Getenv("ERA_MINIO_SECRET_KEY")
	bucket := env("ERA_DRIVE_BUCKET", "era-drive")
	secure := os.Getenv("ERA_MINIO_SECURE") == "1"
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	return &MinIO{client: client, bucket: bucket}, nil
}

func (m *MinIO) Put(data []byte) (string, error) {
	key := "drive/" + uuid.NewString()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := m.client.PutObject(ctx, m.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	return key, err
}

func (m *MinIO) Get(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	b, err := io.ReadAll(obj)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (m *MinIO) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := m.client.ListBuckets(ctx)
	return err
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// Adapter wraps blobstore.Store as drive.BlobStore.
type Adapter struct {
	S Store
}

func (a Adapter) Put(data []byte) (string, error) {
	if a.S == nil {
		return "", fmt.Errorf("blobstore: nil")
	}
	return a.S.Put(data)
}

func (a Adapter) Get(key string) ([]byte, error) {
	if a.S == nil {
		return nil, fmt.Errorf("blobstore: nil")
	}
	return a.S.Get(key)
}
