package tenant

import "testing"

func TestTenantDomainResolve(t *testing.T) {
	s := NewStore()
	if err := s.PutTenant(Tenant{ID: "t1", Name: "Gov Contour", Slug: "gov"}); err != nil {
		t.Fatal(err)
	}
	if err := s.PutDomain(Domain{ID: "d1", TenantID: "t1", FQDN: "mail.gov.az", Primary: true}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ResolveByDomain("mail.gov.az")
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "gov" {
		t.Fatalf("unexpected tenant: %+v", got)
	}
}

func TestDuplicateSlug(t *testing.T) {
	s := NewStore()
	_ = s.PutTenant(Tenant{ID: "t1", Name: "A", Slug: "acme"})
	err := s.PutTenant(Tenant{ID: "t2", Name: "B", Slug: "acme"})
	if err != ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestPutDomainUnknownTenant(t *testing.T) {
	s := NewStore()
	err := s.PutDomain(Domain{ID: "d1", TenantID: "missing", FQDN: "x.local"})
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
