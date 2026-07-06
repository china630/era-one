// Package manifest — загрузка и валидация products.yaml (ADR-0024).
package manifest

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProductsFile — корневая структура products.yaml.
type ProductsFile struct {
	SchemaVersion string         `yaml:"schema_version"`
	Brand         Brand          `yaml:"brand"`
	SharedPlatform SharedPlatform `yaml:"shared_platform"`
	Products      map[string]ProductEntry `yaml:"products"`
	Bundles       map[string]BundleEntry  `yaml:"bundles"`
}

type Brand struct {
	Name             string `yaml:"name"`
	Tagline          string `yaml:"tagline"`
	Domain           string `yaml:"domain"`
	EditionsManifest string `yaml:"editions_manifest"`
}

type SharedPlatform struct {
	Packages []string `yaml:"packages"`
	Services []string `yaml:"services"`
	Docs     string   `yaml:"docs"`
}

type ProductEntry struct {
	Title         string   `yaml:"title"`
	Tagline       string   `yaml:"tagline"`
	Status        string   `yaml:"status"`
	Description   string   `yaml:"description"`
	Agent         bool     `yaml:"agent"`
	PricingModel  string   `yaml:"pricing_model"`
	EditionsRef   string   `yaml:"editions_ref"`
	DeployProfile string   `yaml:"deploy_profile"`
	SitePath      string   `yaml:"site_path"`
	ServiceStub   string   `yaml:"service_stub,omitempty"`
	Docs          []string `yaml:"docs"`
}

type BundleEntry struct {
	Title           string   `yaml:"title"`
	Description     string   `yaml:"description"`
	Products        []string `yaml:"products"`
	DeployProfile   string   `yaml:"deploy_profile"`
	SharedPlatform  bool     `yaml:"shared_platform"`
}

// LoadProducts читает и парсит products.yaml.
func LoadProducts(path string) (*ProductsFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("manifest: read %s: %w", path, err)
	}
	var f ProductsFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("manifest: parse: %w", err)
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return &f, nil
}

// Validate проверяет обязательные поля products.yaml.
func (f *ProductsFile) Validate() error {
	if f.SchemaVersion == "" {
		return fmt.Errorf("manifest: schema_version required")
	}
	if f.Brand.Name == "" {
		return fmt.Errorf("manifest: brand.name required")
	}
	required := []string{"era-control", "era-communications", "era-office"}
	for _, key := range required {
		p, ok := f.Products[key]
		if !ok {
			return fmt.Errorf("manifest: missing product %q", key)
		}
		if p.Title == "" || p.EditionsRef == "" || p.DeployProfile == "" {
			return fmt.Errorf("manifest: product %q incomplete", key)
		}
	}
	if len(f.SharedPlatform.Packages) == 0 {
		return fmt.Errorf("manifest: shared_platform.packages required")
	}
	for _, pkg := range f.SharedPlatform.Packages {
		if !strings.HasPrefix(pkg, "platform/") {
			return fmt.Errorf("manifest: shared package must be platform/*: %q", pkg)
		}
	}
	if b, ok := f.Bundles["era-one-full"]; ok {
		if len(b.Products) != 3 {
			return fmt.Errorf("manifest: era-one-full must list 3 products")
		}
	}
	return nil
}

// ProductKeys возвращает ключи продуктов, отсортированные.
func (f *ProductsFile) ProductKeys() []string {
	keys := make([]string, 0, len(f.Products))
	for k := range f.Products {
		keys = append(keys, k)
	}
	return keys
}
