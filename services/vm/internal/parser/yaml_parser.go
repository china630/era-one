package parser

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"era/services/vm/internal/models"

	"gopkg.in/yaml.v3"
)

// ParseTemplate разбирает YAML в структуру Template и выполняет минимальную валидацию.
func ParseTemplate(data []byte) (*models.Template, error) {
	var t models.Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if strings.TrimSpace(t.ID) == "" || len(t.Requests) == 0 {
		return nil, errors.New("invalid template format")
	}
	return &t, nil
}

// LoadTemplatesFromDir рекурсивно загружает все файлы с расширением .yaml из dirPath.
func LoadTemplatesFromDir(dirPath string) ([]*models.Template, error) {
	var templates []*models.Template
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".yaml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		tmpl, err := ParseTemplate(data)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		templates = append(templates, tmpl)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return templates, nil
}
