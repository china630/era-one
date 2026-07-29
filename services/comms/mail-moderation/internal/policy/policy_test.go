package policy_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"era/services/comms/mail-moderation/internal/policy"
)

func testdata(t *testing.T, rel string) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "testdata", rel)
	return root
}

func loadRules(t *testing.T) []policy.Rule {
	t.Helper()
	doc, err := policy.LoadDocument(testdata(t, "rules/novices-external.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return doc.Rules
}

func TestEvaluate_NovicesExternal_Hold(t *testing.T) {
	rules := loadRules(t)
	msg := policy.Message{
		From:         "alex@company.local",
		To:           []string{"client@external.com"},
		Subject:      "Hello",
		SenderGroups: []string{"novices"},
		Authenticated: true,
	}
	ctx := policy.EvalContext{LocalDomains: []string{"company.local"}}
	res := policy.Evaluate(rules, msg, ctx)
	// priority 5 curator-override matches first for novices+external
	if res.Decision != policy.DecisionHold {
		t.Fatalf("want hold, got %s rule=%s", res.Decision, res.RuleID)
	}
}

func TestEvaluate_BypassExecutives(t *testing.T) {
	rules := loadRules(t)
	msg := policy.Message{
		From:         "ceo@company.local",
		To:           []string{"client@external.com"},
		SenderGroups: []string{"novices", "executives"},
	}
	ctx := policy.EvalContext{LocalDomains: []string{"company.local"}}
	res := policy.Evaluate(rules, msg, ctx)
	if res.Decision != policy.DecisionBypass || res.RuleID != "bypass-exec" {
		t.Fatalf("want bypass-exec, got %s/%s", res.Decision, res.RuleID)
	}
}

func TestEvaluate_InternalNoHold(t *testing.T) {
	// Only external_only rules — internal should pass if no match
	doc := policy.Document{Rules: []policy.Rule{{
		ID:       "ext-only",
		Priority: 10,
		Conditions: policy.Conditions{
			SenderGroups: []string{"novices"},
			ExternalOnly: true,
		},
		Moderator: policy.ModeratorSpec{Mode: policy.ModManager},
	}}}
	msg := policy.Message{
		From:         "alex@company.local",
		To:           []string{"bob@company.local"},
		SenderGroups: []string{"novices"},
	}
	res := policy.Evaluate(doc.Rules, msg, policy.EvalContext{LocalDomains: []string{"company.local"}})
	if res.Decision != policy.DecisionPass {
		t.Fatalf("want pass for internal, got %s", res.Decision)
	}
}

func TestEvaluate_Keywords(t *testing.T) {
	doc := policy.Document{Rules: []policy.Rule{{
		ID:       "kw",
		Priority: 1,
		Conditions: policy.Conditions{
			Keywords:     []string{"Договор"},
			ExternalOnly: true,
		},
		Moderator: policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"legal@company.local"}},
	}}}
	msg := policy.Message{
		From:    "a@company.local",
		To:      []string{"x@out.com"},
		Subject: "Re: Договор поставки",
	}
	res := policy.Evaluate(doc.Rules, msg, policy.EvalContext{LocalDomains: []string{"company.local"}})
	if res.Decision != policy.DecisionHold || res.RuleID != "kw" {
		t.Fatalf("want hold kw, got %s/%s", res.Decision, res.RuleID)
	}
}

func TestEvaluate_VIPDomain(t *testing.T) {
	doc := policy.Document{Rules: []policy.Rule{{
		ID:       "vip",
		Priority: 1,
		Conditions: policy.Conditions{
			RecipientDomains: []string{"vip-customer.com"},
		},
		Moderator: policy.ModeratorSpec{Mode: policy.ModLDAPAttr, LDAPAttr: "extensionAttribute1"},
	}}}
	msg := policy.Message{From: "a@c.local", To: []string{"ceo@vip-customer.com"}}
	res := policy.Evaluate(doc.Rules, msg, policy.EvalContext{LocalDomains: []string{"c.local"}})
	if res.Decision != policy.DecisionHold {
		t.Fatalf("want hold, got %s", res.Decision)
	}
}

func TestYAML_RoundTrip(t *testing.T) {
	path := testdata(t, "rules/novices-external.yaml")
	doc, err := policy.LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := policy.MarshalDocument(doc)
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := policy.ParseDocument(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc2.Rules) != len(doc.Rules) {
		t.Fatalf("rules %d vs %d", len(doc2.Rules), len(doc.Rules))
	}
	// ensure file exists for golden path
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestEvaluate_ModeratedRecipient(t *testing.T) {
	doc := policy.Document{Rules: []policy.Rule{{
		ID:       "dl-finance",
		Priority: 1,
		Conditions: policy.Conditions{
			ModeratedRecipients: []string{"finance@company.local"},
		},
		Moderator: policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"cfo@company.local"}},
	}}}
	msg := policy.Message{From: "anyone@company.local", To: []string{"finance@company.local"}}
	res := policy.Evaluate(doc.Rules, msg, policy.EvalContext{LocalDomains: []string{"company.local"}})
	if res.Decision != policy.DecisionHold || res.RuleID != "dl-finance" {
		t.Fatalf("%s %s", res.Decision, res.RuleID)
	}
}

func TestPriority_StaticOverrideWins(t *testing.T) {
	doc := policy.Document{Rules: []policy.Rule{
		{
			ID:       "manager-default",
			Priority: 20,
			Conditions: policy.Conditions{
				SenderGroups: []string{"novices"},
				ExternalOnly: true,
			},
			Moderator: policy.ModeratorSpec{Mode: policy.ModManager},
		},
		{
			ID:       "alex-curator",
			Priority: 5,
			Conditions: policy.Conditions{
				SenderGroups: []string{"novices"},
				ExternalOnly: true,
			},
			Moderator: policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"sergey@company.local"}},
		},
	}}
	msg := policy.Message{
		From:         "alex@company.local",
		To:           []string{"x@out.com"},
		SenderGroups: []string{"novices"},
	}
	res := policy.Evaluate(doc.Rules, msg, policy.EvalContext{LocalDomains: []string{"company.local"}})
	if res.RuleID != "alex-curator" || res.Decision != policy.DecisionHold {
		t.Fatalf("want alex-curator hold, got %s/%s", res.Decision, res.RuleID)
	}
	if res.Rule.Moderator.Mode != policy.ModStatic {
		t.Fatalf("want static moderator")
	}
}
