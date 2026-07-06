package api

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"

	"era/services/update-service/internal/bundle"
)

func TestBundleKindsRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		kind string
		file string
		env  string
	}{
		{bundle.KindSigmaCorpus, "rule.yml", "ERA_SIGMA_CORPUS"},
		{bundle.KindCVEFeed, "cve.json", "ERA_CVE_FEED_DIR"},
		{bundle.KindConnector, "conn.yaml", "ERA_CONNECTOR_DIR"},
		{bundle.KindAIPack, "pack.yaml", "ERA_AI_PACK_DIR"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			dir := t.TempDir()
			_ = os.WriteFile(filepath.Join(dir, tc.file), []byte("content"), 0o644)
			t.Setenv(tc.env, dir)
			t.Setenv("ERA_BUNDLE_KIND", tc.kind)
			srv, err := New(priv, pub)
			if err != nil {
				t.Fatal(err)
			}
			srv.mu.RLock()
			claims := srv.claims
			token := srv.token
			srv.mu.RUnlock()
			got, err := bundle.Verify(token, pub)
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind != tc.kind {
				t.Fatalf("kind=%s", got.Kind)
			}
			if claims.FileCount != got.FileCount || claims.ManifestHash != got.ManifestHash {
				t.Fatalf("claims mismatch: %+v vs %+v", claims, got)
			}
		})
	}
}

func TestValidKind(t *testing.T) {
	if !bundle.ValidKind("cve-feed") || bundle.ValidKind("unknown") {
		t.Fatal("ValidKind")
	}
}
