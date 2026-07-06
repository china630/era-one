package minio

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	mg "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectInfo — объект в бакете (для листинга образов).
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// Lister — абстракция над MinIO list (тестируется mock).
type Lister interface {
	ListObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error)
}

// Client — minio-go обёртка.
type Client struct {
	mc *mg.Client
}

// NewFromEnv создаёт клиент из ERA_MINIO_* env.
func NewFromEnv() (*Client, error) {
	endpoint := strings.TrimPrefix(envOr("ERA_MINIO_ENDPOINT", ""), "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if endpoint == "" {
		return nil, fmt.Errorf("ERA_MINIO_ENDPOINT not set")
	}
	access := envOr("ERA_MINIO_ACCESS_KEY", "era")
	secret := envOr("ERA_MINIO_SECRET_KEY", "era_dev_pw")
	useSSL := strings.HasPrefix(os.Getenv("ERA_MINIO_ENDPOINT"), "https://")
	mc, err := mg.New(endpoint, &mg.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Client{mc: mc}, nil
}

func (c *Client) ListObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	if c == nil || c.mc == nil {
		return nil, fmt.Errorf("minio client nil")
	}
	ch := c.mc.ListObjects(ctx, bucket, mg.ListObjectsOptions{Prefix: prefix, Recursive: true})
	var out []ObjectInfo
	for obj := range ch {
		if obj.Err != nil {
			return nil, obj.Err
		}
		if strings.HasSuffix(obj.Key, "/") {
			continue
		}
		out = append(out, ObjectInfo{
			Key: obj.Key, Size: obj.Size, LastModified: obj.LastModified,
		})
	}
	return out, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return strings.TrimSpace(v)
	}
	return def
}
