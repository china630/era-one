package deception

import "testing"

func TestHoneytokenGolden(t *testing.T) {
	cases := []struct {
		payload string
		want    bool
		id      string
	}{
		{`{"file":"\\\\decoy-share\\finance.xlsx"}`, true, "era-deception-honeytoken"},
		{`{"user":"alice","action":"read"}`, false, ""},
	}
	for _, tc := range cases {
		ok, hit := MatchHoneytoken(tc.payload)
		if ok != tc.want {
			t.Fatalf("payload=%q ok=%v want=%v", tc.payload, ok, tc.want)
		}
		if tc.want && hit.RuleID != tc.id {
			t.Fatalf("rule=%s", hit.RuleID)
		}
	}
}
