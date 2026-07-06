package build

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"era/services/update-service/internal/bundle"
)

// ScanDir строит манифест из каталога с заданными расширениями.
func ScanDir(root string, exts ...string) (*bundle.Manifest, error) {
	if root == "" {
		return &bundle.Manifest{}, nil
	}
	extSet := make(map[string]bool, len(exts))
	for _, e := range exts {
		extSet[strings.ToLower(e)] = true
	}
	var files []bundle.FileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if len(extSet) > 0 && !extSet[ext] {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		files = append(files, bundle.FileEntry{
			Path:   rel,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   int64(len(b)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &bundle.Manifest{Files: files}, nil
}

// ScanCorpus строит манифест из каталога sigma-corpus.
func ScanCorpus(root string) (*bundle.Manifest, error) {
	if root == "" {
		root = "data/sigma-corpus/curated"
	}
	var files []bundle.FileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".yml") && !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		files = append(files, bundle.FileEntry{
			Path:   rel,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   int64(len(b)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &bundle.Manifest{Files: files}, nil
}

// WriteOfflineBundle пишет подписанный токен в файл (air-gap dual-use).
func WriteOfflineBundle(path, token string) error {
	return os.WriteFile(path, []byte(token+"\n"), 0o644)
}
