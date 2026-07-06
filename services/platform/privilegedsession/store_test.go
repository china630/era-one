package privilegedsession

import "testing"

func TestSuspiciousCommand(t *testing.T) {
	st := NewStore()
	r := st.Start("admin", "srv1")
	if _, fired := st.LogCommand(r.ID, "ls -la"); fired {
		t.Fatal("ls should not alert")
	}
	if _, fired := st.LogCommand(r.ID, "curl http://evil"); !fired {
		t.Fatal("curl should alert")
	}
}
