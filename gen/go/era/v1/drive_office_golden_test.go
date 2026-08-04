package erav1_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	erav1 "era/contracts/gen/era/v1"
	"google.golang.org/protobuf/proto"
)

func TestDriveServiceRegistered(t *testing.T) {
	if erav1.DriveService_ServiceDesc.ServiceName != "era.v1.DriveService" {
		t.Fatalf("unexpected service: %s", erav1.DriveService_ServiceDesc.ServiceName)
	}
}

func TestDriveObjectWireGolden(t *testing.T) {
	obj := &erav1.DriveObject{
		Id:            "obj-1",
		TenantId:      "t-demo",
		FolderId:      "",
		Name:          "memo.erad",
		SizeBytes:     42,
		ContentType:   "application/vnd.era.erad",
		ContentFormat: erav1.DriveContentFormat_DRIVE_CONTENT_FORMAT_ERAD,
		Version:       1,
		BlobKey:       "drive/abc",
		Acl: &erav1.DriveACL{
			TenantId: "t-demo",
			Entries: []*erav1.DriveACLEntry{
				{Principal: "user:alice", Role: erav1.DriveACLRole_DRIVE_ACL_ROLE_OWNER},
			},
		},
	}
	got, err := proto.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	goldenPath := filepath.Join("testdata", "drive_object.golden.bin")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.Getenv("ERA_UPDATE_GOLDEN") == "1" {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("updated golden %s", goldenPath)
			return
		}
		t.Fatalf("read golden (set ERA_UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("drive object wire mismatch; set ERA_UPDATE_GOLDEN=1 to update")
	}
	var decoded erav1.DriveObject
	if err := proto.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.GetName() != "memo.erad" || decoded.GetContentFormat() != erav1.DriveContentFormat_DRIVE_CONTENT_FORMAT_ERAD {
		t.Fatalf("roundtrip fields: name=%q fmt=%v", decoded.GetName(), decoded.GetContentFormat())
	}
}

func TestEradDocumentWireGolden(t *testing.T) {
	doc := &erav1.EradDocument{
		Id:            "doc-1",
		TenantId:      "t-demo",
		DriveObjectId: "obj-1",
		Format:        erav1.DocumentFormat_DOCUMENT_FORMAT_ERAD,
		Blocks: []*erav1.Block{
			{
				Id: "b1",
				Attrs: &erav1.BlockAttrs{
					BlockType: erav1.BlockType_BLOCK_TYPE_PARAGRAPH,
				},
				Inlines: []*erav1.InlineSpan{{Text: "Salam", Bold: true}},
			},
		},
	}
	got, err := proto.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	goldenPath := filepath.Join("testdata", "erad_document.golden.bin")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.Getenv("ERA_UPDATE_GOLDEN") == "1" {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("updated golden %s", goldenPath)
			return
		}
		t.Fatalf("read golden (set ERA_UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("erad document wire mismatch; set ERA_UPDATE_GOLDEN=1 to update")
	}
}

func TestEratSheetWireGolden(t *testing.T) {
	sheet := &erav1.EratSheet{
		Id:            "sheet-1",
		TenantId:      "t-demo",
		DriveObjectId: "obj-sheet-1",
		Format:        erav1.DocumentFormat_DOCUMENT_FORMAT_ERAT,
		Name:          "budget.erat",
		Rows:          1024,
		Cols:          256,
		Cells: []*erav1.EratCell{
			{Addr: "A1", Value: "10"},
			{Addr: "A2", Value: "20"},
			{Addr: "A3", Formula: "=SUM(A1:A2)", Value: "30"},
		},
	}
	got, err := proto.Marshal(sheet)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	goldenPath := filepath.Join("testdata", "erat_sheet.golden.bin")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.Getenv("ERA_UPDATE_GOLDEN") == "1" {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("updated golden %s", goldenPath)
			return
		}
		t.Fatalf("read golden (set ERA_UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("erat sheet wire mismatch; set ERA_UPDATE_GOLDEN=1 to update")
	}
	var decoded erav1.EratSheet
	if err := proto.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.GetName() != "budget.erat" || decoded.GetFormat() != erav1.DocumentFormat_DOCUMENT_FORMAT_ERAT {
		t.Fatalf("roundtrip: name=%q fmt=%v", decoded.GetName(), decoded.GetFormat())
	}
}

func TestErapDeckWireGolden(t *testing.T) {
	deck := &erav1.ErapDeck{
		Id:            "deck-1",
		TenantId:      "t-demo",
		DriveObjectId: "obj-deck-1",
		Format:        erav1.DocumentFormat_DOCUMENT_FORMAT_ERAP,
		Name:          "brief.erap",
		Slides: []*erav1.ErapSlide{
			{Id: "s1", Title: "Agenda", Body: "Item one"},
		},
	}
	got, err := proto.Marshal(deck)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	goldenPath := filepath.Join("testdata", "erap_deck.golden.bin")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.Getenv("ERA_UPDATE_GOLDEN") == "1" {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("updated golden %s", goldenPath)
			return
		}
		t.Fatalf("read golden (set ERA_UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("erap deck wire mismatch; set ERA_UPDATE_GOLDEN=1 to update")
	}
}
