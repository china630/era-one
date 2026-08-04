//go:build integration

package repo_test

import (
	"bytes"
	"os"
	"testing"

	"era/services/comms/mail/internal/blobstore"
	"era/services/comms/mail/internal/repo"
)

func TestPostgresBlobOffload(t *testing.T) {
	endpoint := os.Getenv("ERA_MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:9000"
		os.Setenv("ERA_MINIO_ENDPOINT", endpoint)
	}
	if os.Getenv("ERA_MINIO_ACCESS_KEY") == "" {
		os.Setenv("ERA_MINIO_ACCESS_KEY", "era")
	}
	if os.Getenv("ERA_MINIO_SECRET_KEY") == "" {
		os.Setenv("ERA_MINIO_SECRET_KEY", "era_ci_pw")
	}
	if os.Getenv("ERA_MINIO_BUCKET") == "" {
		os.Setenv("ERA_MINIO_BUCKET", "era-comms")
	}

	dsn := pgDSN()
	pg, err := repo.OpenPostgres(dsn)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	defer pg.Close()

	if err := pg.PingBlob(); err != nil {
		t.Skipf("minio unavailable: %v", err)
	}

	email := "blob-int-" + t.Name() + "@mail.gov.az"
	_, _ = pg.CreateMailbox("t-demo", email, "blob-pass", 64<<20)

	threshold := blobstore.ThresholdBytes()
	body := bytes.Repeat([]byte("B"), threshold+1024)
	msg, err := pg.DeliverRaw(email, body, "sender@mail.gov.az")
	if err != nil {
		t.Fatal(err)
	}
	if msg.MinioKey == "" {
		t.Fatal("expected minio_key offload for large message")
	}

	msgs, err := pg.ListMessages(email)
	if err != nil {
		t.Fatal(err)
	}
	var found *repo.Message
	for _, m := range msgs {
		if m != nil && m.ID == msg.ID {
			found = m
			break
		}
	}
	if found == nil {
		t.Fatalf("expected delivered message id=%d among %d", msg.ID, len(msgs))
	}
	if !bytes.Equal(found.Raw, body) {
		t.Fatalf("hydrated body mismatch: got %d bytes want %d", len(found.Raw), len(body))
	}
}
