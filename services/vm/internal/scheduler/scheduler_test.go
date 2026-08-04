package scheduler

import "testing"

func TestCreateAndList(t *testing.T) {
	s := New()
	j := s.Create("nightly", []string{"10.0.0.1"}, "@every 12h", 3)
	if j.ID == "" {
		t.Fatal("missing id")
	}
	if len(s.List()) != 1 {
		t.Fatal("expected one job")
	}
	got, ok := s.Get(j.ID)
	if !ok || got.Name != "nightly" {
		t.Fatal("get failed")
	}
	s.MarkRun(j.ID, "ok")
	got, _ = s.Get(j.ID)
	if got.LastStatus != "ok" {
		t.Fatal("mark run failed")
	}

	en := false
	upd, ok := s.Update(j.ID, UpdateFields{Name: "daily", Enabled: &en})
	if !ok || upd.Name != "daily" || upd.Enabled {
		t.Fatalf("update failed: %+v", upd)
	}
	if !s.Delete(j.ID) || len(s.List()) != 0 {
		t.Fatal("delete failed")
	}
}
