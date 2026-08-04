package audit

import "testing"

func TestNewFromEnvStrictRequiresCH(t *testing.T) {
	t.Setenv("ERA_MAIL_AUDIT_REQUIRE", "1")
	t.Setenv("ERA_CH_ADDR", "")
	if _, err := NewFromEnvStrict(); err == nil {
		t.Fatal("expected error when audit required without CH")
	}
	t.Setenv("ERA_MAIL_AUDIT_REQUIRE", "0")
	w, err := NewFromEnvStrict()
	if err != nil {
		t.Fatal(err)
	}
	if w.IsConfigured() {
		t.Fatal("expected noop")
	}
}
