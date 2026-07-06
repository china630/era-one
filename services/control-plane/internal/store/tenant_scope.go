// Tenant-scoped queries при ERA_MULTI_TENANT=1 (S7-11).
package store

import "os"

// MultiTenantEnabled — true если ERA_MULTI_TENANT=1.
func MultiTenantEnabled() bool {
	return os.Getenv("ERA_MULTI_TENANT") == "1"
}

// Scoped возвращает store с фильтром tenant_id для list-запросов.
func Scoped(r Repository, tenantID string) Repository {
	if !MultiTenantEnabled() || tenantID == "" {
		return r
	}
	switch s := r.(type) {
	case *sqliteStore:
		cp := *s
		cp.tenantFilter = tenantID
		return &cp
	case *postgresStore:
		cp := *s
		cp.tenantFilter = tenantID
		return &cp
	case *memoryStore:
		cp := *s
		cp.tenantFilter = tenantID
		return &cp
	default:
		return r
	}
}
