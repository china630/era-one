package checkout

import (
	"os"
	"testing"
)

func TestApproverLDAPAllowlist(t *testing.T) {
	t.Setenv("ERA_LDAP_APPROVER_GROUPS", "pam-approvers,security-admins")
	if ApproverInLDAPGroups([]string{"guest"}) {
		t.Fatal("guest should be denied")
	}
	if !ApproverInLDAPGroups([]string{"pam-approvers"}) {
		t.Fatal("pam-approvers allowed")
	}
	if !ApproverInLDAPGroups(ParseGroups("security-admins,other")) {
		t.Fatal("security-admins allowed")
	}
	os.Unsetenv("ERA_LDAP_APPROVER_GROUPS")
	if !ApproverInLDAPGroups(nil) {
		t.Fatal("empty env allows role-only flow")
	}
}

func TestApproverDeniedForWrongGroup(t *testing.T) {
	t.Setenv("ERA_LDAP_APPROVER_GROUPS", "pam-approvers")
	if ApproverInLDAPGroups([]string{"analyst"}) {
		t.Fatal("analyst should not be in approver allowlist")
	}
}
