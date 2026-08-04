package drive_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	erav1 "era/contracts/gen/era/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestDriveACLGoldenJSON(t *testing.T) {
	msg := &erav1.DriveACL{
		TenantId: "t-demo",
		Entries: []*erav1.DriveACLEntry{
			{Principal: "user:u-owner", Role: erav1.DriveACLRole_DRIVE_ACL_ROLE_OWNER},
			{Principal: "tenant:members", Role: erav1.DriveACLRole_DRIVE_ACL_ROLE_READ},
			{Principal: "group:finance", Role: erav1.DriveACLRole_DRIVE_ACL_ROLE_WRITE},
		},
	}
	got, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "drive_acl.golden.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v (run with -update)", err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		var gotMsg, wantMsg erav1.DriveACL
		if err := protojson.Unmarshal(bytes.TrimSpace(got), &gotMsg); err != nil {
			t.Fatal(err)
		}
		if err := protojson.Unmarshal(bytes.TrimSpace(want), &wantMsg); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(&gotMsg, &wantMsg) {
			t.Fatalf("ACL golden mismatch; run with -update if intentional\n got: %s\nwant: %s", got, want)
		}
	}
}
