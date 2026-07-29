package mail_test

import (
	"testing"

	"era/ui/mail"
)

func TestDocumentsEditLink(t *testing.T) {
	c := mail.NewDocumentsClient("https://app.test.local")
	link, err := c.EditLink("obj-123")
	if err != nil {
		t.Fatal(err)
	}
	if link != "https://app.test.local/docs/obj-123" {
		t.Fatalf("link %q", link)
	}
}

func TestDocumentsMimeGate(t *testing.T) {
	if !mail.IsEradOrDocx("memo.docx", "") {
		t.Fatal("docx expected")
	}
	if !mail.IsEradOrDocx("doc.erad", "application/vnd.era.erad") {
		t.Fatal("erad expected")
	}
	if mail.IsEradOrDocx("image.png", "image/png") {
		t.Fatal("png should not match")
	}
}

func TestDocumentsClientNil(t *testing.T) {
	var c *mail.DocumentsClient
	if _, err := c.EditLink("x"); err == nil {
		t.Fatal("expected error")
	}
}
