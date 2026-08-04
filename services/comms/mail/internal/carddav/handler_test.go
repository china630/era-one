package carddav

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/comms/mail/internal/repo"
)

func TestCardDAVSyncCollectionGolden(t *testing.T) {
	r := repo.NewMemory()
	_, _ = r.CreateMailbox("t1", "alice@mail.gov.az", "pw", 1<<20)
	_, _ = r.PutContact("alice@mail.gov.az", "c1", "BEGIN:VCARD\nFN:Alice\nEND:VCARD")
	_, _ = r.PutContact("alice@mail.gov.az", "c2", "BEGIN:VCARD\nFN:Bob\nEND:VCARD")
	h := &Handler{Repo: r}

	req := httptest.NewRequest("REPORT", "/carddav/alice@mail.gov.az/", strings.NewReader(`<?xml version="1.0"?>
<d:sync-collection xmlns:d="DAV:">
  <d:sync-token>0</d:sync-token>
  <d:sync-level>1</d:sync-level>
  <d:prop><d:getetag/></d:prop>
</d:sync-collection>`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !bytes.Contains([]byte(body), []byte("c1.vcf")) || !bytes.Contains([]byte(body), []byte("c2.vcf")) {
		t.Fatalf("missing contacts %s", body)
	}
	if !bytes.Contains([]byte(body), []byte("<d:sync-token>2</d:sync-token>")) {
		t.Fatalf("sync-token %s", body)
	}

	req = httptest.NewRequest("REPORT", "/carddav/alice@mail.gov.az/", strings.NewReader(`<?xml version="1.0"?>
<d:sync-collection xmlns:d="DAV:">
  <d:sync-token>2</d:sync-token>
  <d:sync-level>1</d:sync-level>
  <d:prop><d:getetag/></d:prop>
</d:sync-collection>`))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), "c1.vcf") {
		t.Fatalf("expected no changes after sync %s", rec.Body.String())
	}
}

func TestCardDAVPropfindMulti(t *testing.T) {
	r := repo.NewMemory()
	_, _ = r.PutContact("alice@mail.gov.az", "c1", "BEGIN:VCARD\nFN:One\nEND:VCARD")
	_, _ = r.PutContact("alice@mail.gov.az", "c2", "BEGIN:VCARD\nFN:Two\nEND:VCARD")
	h := &Handler{Repo: r}
	req := httptest.NewRequest("PROPFIND", "/carddav/alice@mail.gov.az/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if strings.Count(rec.Body.String(), "<d:response>") < 2 {
		t.Fatalf("body %s", rec.Body.String())
	}
}
