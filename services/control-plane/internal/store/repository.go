// Package store — persistent хранилище control-plane (GA Wave-1).
package store

import (
	"fmt"
	"os"
	"time"
)

// Asset — зарегистрированный хост.
// Полная CMDB-модель — cmdb_types.go (ITAM fields).

// Case — инцидент SOC.
type Case struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	TenantID     string    `json:"tenant_id,omitempty"`
	Assignee    string    `json:"assignee,omitempty"`
	DetectionID string    `json:"detection_id,omitempty"`
	NodeID      string    `json:"node_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PolicyBundle — ссылки на политики детекции.
type PolicyBundle struct {
	Version string            `json:"version"`
	Rules   map[string]string `json:"rules_ref"`
}

// Repository — assets, cases, policy (SQLite или memory).
type Repository interface {
	CmdbOps
	EnforcementOps
	DeployOps
	CaseOps
	HybridOps
	UpsertAsset(a *Asset)
	ListAssets() []*Asset
	AssetCoverage() float64
	CreateCase(c *Case)
	GetCase(id string) (*Case, bool)
	UpdateCase(id string, fn func(*Case)) (*Case, bool)
	ListCases() []*Case
	Policy() PolicyBundle
	SetPolicy(PolicyBundle)
	Close() error
}

// NewFromEnv создаёт store: ERA_STORE_DRIVER=postgres|sqlite, ERA_STORE_PATH, ERA_STORE_DSN.
func NewFromEnv() (Repository, error) {
	switch os.Getenv("ERA_STORE_DRIVER") {
	case "postgres":
		dsn := os.Getenv("ERA_STORE_DSN")
		if dsn == "" {
			return nil, fmt.Errorf("ERA_STORE_DSN required for postgres driver")
		}
		return NewPostgres(dsn)
	default:
		path := os.Getenv("ERA_STORE_PATH")
		if path == "" {
			return NewMemory(), nil
		}
		return NewSQLite(path)
	}
}

// NewMemory — dev/fallback.
func NewMemory() Repository {
	return newMemoryStore()
}

// DefaultPolicy для новых инсталляций.
func DefaultPolicy() PolicyBundle {
	return PolicyBundle{
		Version: "3.0.0-ga",
		Rules: map[string]string{
			"sigma":    "data/sigma-corpus",
			"curated":  "data/sigma-corpus/curated",
			"national": "data/national-iocs",
		},
	}
}

func validateStorePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty store path")
	}
	return nil
}
