package rules_test

import (
	"testing"

	"era/services/comms/mail-moderation/internal/policy"
	"era/services/comms/mail-moderation/internal/rules"
)

func TestMemorySave_RoundTrip(t *testing.T) {
	m := &rules.MemorySave{}
	doc := policy.Document{Rules: []policy.Rule{{ID: "r1", Priority: 1, Moderator: policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"a@b.c"}}}}}
	if err := m.SaveDocument(doc); err != nil {
		t.Fatal(err)
	}
	got, err := m.LoadDocument()
	if err != nil || len(got.Rules) != 1 || got.Rules[0].ID != "r1" {
		t.Fatalf("%v %v", got, err)
	}
}
