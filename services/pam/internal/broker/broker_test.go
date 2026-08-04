package broker

import "testing"

func TestBindNoPasswordInPublicMeta(t *testing.T) {
	s := NewStore()
	inj, err := s.Bind("sess1", "admin", "s3cret!", "sec-1", "co-1")
	if err != nil {
		t.Fatal(err)
	}
	m := inj.PublicMeta()
	if m["password"] != nil {
		t.Fatal("password leaked")
	}
	if m["inject_token"] == "" || m["username"] != "admin" {
		t.Fatalf("%v", m)
	}
	if !s.HasPassword("sess1") {
		t.Fatal("expected password held")
	}
	u, p, ok := s.ConsumePassword("sess1")
	if !ok || u != "admin" || p != "s3cret!" {
		t.Fatalf("%s %s %v", u, p, ok)
	}
	if s.HasPassword("sess1") {
		t.Fatal("password should be one-shot")
	}
}
