package atlas

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadFromDir loads the first *.json Atlas pack under dir (update-service / USB air-gap).
func (s *Store) LoadFromDir(dir string) (Pack, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Pack{}, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		path := filepath.Join(dir, name)
		if err := s.LoadFile(path); err != nil {
			return Pack{}, fmt.Errorf("%s: %w", path, err)
		}
		return s.Meta(), nil
	}
	return Pack{}, fmt.Errorf("no atlas json pack in %s", dir)
}

// VerifyUnsignedManifest checks a local unsigned pack file parses (air-gap USB).
func VerifyUnsignedManifest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var p Pack
	return json.Unmarshal(data, &p)
}
