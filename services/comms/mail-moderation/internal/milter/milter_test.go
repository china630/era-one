package milter_test

import (
	"testing"

	"era/services/comms/mail-moderation/internal/milter"
)

func TestStub(t *testing.T) {
	d := milter.Stub(true, "policy")
	if d.Action != milter.ActionQuarantine {
		t.Fatal(d)
	}
	d = milter.Stub(false, "")
	if d.Action != milter.ActionAccept {
		t.Fatal(d)
	}
}
