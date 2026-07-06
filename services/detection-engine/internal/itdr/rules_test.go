package itdr

import "testing"

func TestKerberoasting(t *testing.T) {
	payload := `{"auth_type":"kerberos","event":"TGS-REQ","etype":23,"spn":"MSSQLSvc/db.corp"}`
	ok, r := MatchAuth(payload)
	if !ok || r.ID != "era-itdr-kerberoasting" {
		t.Fatalf("expected kerberoasting, ok=%v rule=%s", ok, r.ID)
	}
}

func TestDCSync(t *testing.T) {
	payload := `{"event":"GetChangesAll","api":"drsuapi","subject":"svc-repl"}`
	ok, r := MatchAuth(payload)
	if !ok || r.ID != "era-itdr-dcsync" {
		t.Fatalf("expected dcsync, ok=%v rule=%s", ok, r.ID)
	}
}

func TestGoldenTicket(t *testing.T) {
	payload := `{"principal":"krbtgt","ticket_lifetime_hours":99999,"event":"TGT issue"}`
	ok, r := MatchAuth(payload)
	if !ok || r.ID != "era-itdr-golden-ticket" {
		t.Fatalf("expected golden ticket, ok=%v rule=%s", ok, r.ID)
	}
}
