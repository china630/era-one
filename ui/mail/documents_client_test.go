package mail_test

import (
	"strings"
	"testing"

	"era/ui/mail"
)

func TestDocumentsEditLink(t *testing.T) {
	c := mail.NewDocumentsClient("https://app.test.local")
	c.LicenseOK = true
	c.IntentSecret = []byte("intent-secret")
	link, err := c.EditLink("obj-123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(link, "https://app.test.local/docs/obj-123") {
		t.Fatalf("link %q", link)
	}
	if !strings.Contains(link, "intent_sig=") || !strings.Contains(link, "intent_exp=") {
		t.Fatalf("expected signed intent in %q", link)
	}
}

func TestDocumentsEditLinkUnlicensed(t *testing.T) {
	c := &mail.DocumentsClient{
		WorkspaceBaseURL: "https://app.test.local",
		IntentSecret:     []byte("s"),
		LicenseOK:        false,
	}
	if _, err := c.EditLink("obj"); err == nil {
		t.Fatal("expected license error")
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

func TestVerifyIntent(t *testing.T) {
	secret := []byte("s")
	c := &mail.DocumentsClient{WorkspaceBaseURL: "https://x", IntentSecret: secret, LicenseOK: true}
	link, err := c.EditLink("oid")
	if err != nil {
		t.Fatal(err)
	}
	u := link[strings.Index(link, "?"):]
	params := map[string]string{}
	for _, p := range strings.Split(strings.TrimPrefix(u, "?"), "&") {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	if !mail.VerifyIntent(secret, "oid", params["intent_exp"], params["intent_sig"]) {
		t.Fatal("verify failed")
	}
}
