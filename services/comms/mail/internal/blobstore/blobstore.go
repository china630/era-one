// Package blobstore — MinIO/S3 blob storage for mail bodies (R2-B).
package blobstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Store uploads and downloads message blobs.
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
	bucket := env("ERA_MINIO_BUCKET", "era-comms")
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
	key := "mail/" + uuid.NewString()
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
	return io.ReadAll(obj)
}

func (m *MinIO) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := m.client.ListBuckets(ctx)
	return err
}

// ThresholdBytes returns blob offload threshold from env.
func ThresholdBytes() int {
	if v := os.Getenv("ERA_MAIL_BLOB_THRESHOLD_BYTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 256 * 1024
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// Nop is a no-op store for tests.
type Nop struct{}

func (Nop) Put(data []byte) (string, error) { return "", fmt.Errorf("blob store disabled") }
func (Nop) Get(string) ([]byte, error)      { return nil, fmt.Errorf("blob store disabled") }
func (Nop) Ping() error                     { return nil }
