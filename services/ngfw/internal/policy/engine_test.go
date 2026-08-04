package policy

import "testing"

func TestDenyExternalSMB(t *testing.T) {
	e := Default()
	d := e.Evaluate("203.0.113.5", "10.0.0.5", "tcp", 445)
	if d.Allowed {
		t.Fatalf("expected deny, got %+v", d)
	}
	if d.RuleID != "deny-external-smb" {
		t.Fatal(d.RuleID)
	}
}

func TestAllowInternal(t *testing.T) {
	e := Default()
	d := e.Evaluate("10.0.0.12", "10.0.0.50", "tcp", 443)
	if !d.Allowed {
		t.Fatalf("expected allow, got %+v", d)
	}
}

func TestEvaluateGolden(t *testing.T) {
	e := Default()
	type row struct {
		Src, Dst, Proto string
		Port            uint32
		Allow           bool
		Rule            string
	}
	cases := []row{
		{"203.0.113.1", "8.8.8.8", "tcp", 445, false, "deny-external-smb"},
		{"10.1.1.1", "10.2.2.2", "tcp", 22, true, "allow-internal"},
		{"203.0.113.9", "8.8.8.8", "tcp", 443, true, "default-allow"},
	}
	for _, c := range cases {
		d := e.Evaluate(c.Src, c.Dst, c.Proto, c.Port)
		if d.Allowed != c.Allow || d.RuleID != c.Rule {
			t.Fatalf("%+v => %+v", c, d)
		}
	}
}

func TestReplacePersist(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/policy.json"
	e := Default()
	e.SetPath(path)
	rules := []Rule{{ID: "deny-all-ssh", Action: ActionDeny, DstPort: 22, SrcCIDR: "0.0.0.0/0"}}
	if err := e.Replace(rules); err != nil {
		t.Fatal(err)
	}
	e2 := &Engine{}
	if err := e2.Load(path); err != nil {
		t.Fatal(err)
	}
	d := e2.Evaluate("1.1.1.1", "2.2.2.2", "tcp", 22)
	if d.Allowed || d.RuleID != "deny-all-ssh" {
		t.Fatalf("%+v", d)
	}
}
