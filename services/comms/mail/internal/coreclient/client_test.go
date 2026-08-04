package coreclient

import "testing"

func TestStubStatus(t *testing.T) {
	c := NewStub()
	st := c.Status()
	if st.SMTPReady || st.IMAPReady {
		t.Fatalf("stub should report not ready: %+v", st)
	}
	if st.Version == "" {
		t.Fatal("version required")
	}
}
