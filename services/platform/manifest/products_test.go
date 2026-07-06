package manifest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func TestLoadProductsFromRepo(t *testing.T) {
	path := filepath.Join(repoRoot(t), "products.yaml")
	f, err := LoadProducts(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Brand.Name != "ERA One" {
		t.Fatalf("brand: %q", f.Brand.Name)
	}
	ctrl := f.Products["era-control"]
	if !ctrl.Agent || ctrl.Status != "ga" {
		t.Fatalf("era-control: %+v", ctrl)
	}
	comms := f.Products["era-communications"]
	if comms.Status != "roadmap" {
		t.Fatalf("era-communications status: %q", comms.Status)
	}
	if _, err := os.Stat(filepath.Join(repoRoot(t), comms.DeployProfile)); err != nil {
		t.Fatalf("comms deploy profile: %v", err)
	}
}

func TestValidateRejectsMissingProduct(t *testing.T) {
	f := &ProductsFile{
		SchemaVersion: "1.0",
		Brand:         Brand{Name: "ERA One"},
		SharedPlatform: SharedPlatform{Packages: []string{"platform/identity"}},
		Products: map[string]ProductEntry{
			"era-control": {Title: "X", EditionsRef: "e.yaml", DeployProfile: "d.yaml"},
		},
	}
	if err := f.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
