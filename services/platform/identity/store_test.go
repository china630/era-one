package identity

import "testing"

func TestStoreUserAndPermissions(t *testing.T) {
	s := NewStore()
	if err := s.PutRole(Role{
		ID: "role-admin", TenantID: "t1", Name: "admin",
		Permissions: []string{"control:read", "comms:read"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.PutUser(User{
		ID: "u1", TenantID: "t1", Email: "admin@contour.local",
		DisplayName: "Admin", Active: true, RoleIDs: []string{"role-admin"},
	}); err != nil {
		t.Fatal(err)
	}
	perms, err := s.PermissionsForUser("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(perms) != 2 {
		t.Fatalf("expected 2 permissions, got %v", perms)
	}
	got, err := s.GetUser("u1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "admin@contour.local" {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestStoreTenantRequired(t *testing.T) {
	s := NewStore()
	if err := s.PutUser(User{ID: "u1"}); err != ErrTenantMissing {
		t.Fatalf("expected ErrTenantMissing, got %v", err)
	}
}

func TestListUsersByTenant(t *testing.T) {
	s := NewStore()
	_ = s.PutUser(User{ID: "u1", TenantID: "t1", Email: "a@x"})
	_ = s.PutUser(User{ID: "u2", TenantID: "t2", Email: "b@x"})
	list := s.ListUsersByTenant("t1")
	if len(list) != 1 || list[0].ID != "u1" {
		t.Fatalf("unexpected list: %+v", list)
	}
}
