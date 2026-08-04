package apply

import (
	"testing"

	"era/services/ngfw/internal/policy"
)

func TestSelectDefaultNoop(t *testing.T) {
	t.Setenv("ERA_NGFW_APPLY", "")
	b := Select()
	if b.Name() != "noop" {
		t.Fatalf("want noop got %s", b.Name())
	}
	if err := b.ApplyDeny(policy.Rule{DstPort: 445}); err != nil {
		t.Fatal(err)
	}
}

func TestDryRun(t *testing.T) {
	s := DryRun(policy.Rule{DstPort: 445, SrcCIDR: "0.0.0.0/0", DstCIDR: "10.0.0.1/32"})
	if s == "" {
		t.Fatal("empty dry-run")
	}
}
