package ews

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"era/services/comms/calendar/store"
	"era/services/comms/mail/internal/repo"
)

func TestCreateItemGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/create_item_request.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	r := repo.NewMemory()
	_, _ = r.CreateMailbox("t1", "alice@mail.gov.az", "pw", 1<<20)
	h := &Handler{Repo: r, Cal: store.New()}
	req := httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", bytes.NewReader(raw))
	req.Header.Set("SOAPAction", "CreateItem")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("msg-")) {
		t.Fatalf("response %s", rec.Body.String())
	}
	msgs, _ := r.ListMessages("alice@mail.gov.az")
	if len(msgs) != 1 {
		t.Fatalf("messages %d", len(msgs))
	}
}

func TestFindFolderGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/find_folder_request.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{Repo: repo.NewMemory(), Cal: store.New()}
	req := httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", bytes.NewReader(raw))
	req.Header.Set("SOAPAction", "FindFolder")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("FindFolderResponse")) {
		t.Fatalf("body %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Inbox")) {
		t.Fatalf("missing inbox %s", rec.Body.String())
	}
}

func TestDeleteItemGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/delete_item_request.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{Repo: repo.NewMemory(), Cal: store.New()}
	req := httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", bytes.NewReader(raw))
	req.Header.Set("SOAPAction", "DeleteItem")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("DeleteItemResponseMessage")) {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestSyncFolderItemsMultiGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/sync_folder_items_request.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	cal := store.New()
	cal.Put("alice@mail.gov.az", "evt-ews-1", `BEGIN:VEVENT
UID:evt-ews-1
SUMMARY:Standup
DTSTART:20260707T090000Z
END:VEVENT`)
	r := repo.NewMemory()
	_, _ = r.CreateMailbox("t1", "alice@mail.gov.az", "pw", 1<<20)
	_, _ = r.AddEWSMessage("alice@mail.gov.az", "Mail A", "body")
	_, _ = r.AddEWSMessage("alice@mail.gov.az", "Mail B", "body")
	h := &Handler{Repo: r, Cal: cal}
	req := httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", bytes.NewReader(raw))
	req.Header.Set("SOAPAction", "SyncFolderItems")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("sync %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "evt-ews-1") {
		t.Fatalf("missing calendar %s", body)
	}
	if strings.Count(body, "<Create><Message>") < 2 {
		t.Fatalf("expected multiple mail items %s", body)
	}
}

func TestCreateContactGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/create_contact_request.golden.xml")
	if err != nil {
		t.Fatal(err)
	}
	r := repo.NewMemory()
	_, _ = r.CreateMailbox("t1", "alice@mail.gov.az", "pw", 1<<20)
	h := &Handler{Repo: r, Cal: store.New()}
	req := httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", bytes.NewReader(raw))
	req.Header.Set("SOAPAction", "CreateContact")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("CreateContactResponse")) {
		t.Fatalf("response %s", rec.Body.String())
	}
	contacts, _ := r.ListContacts("alice@mail.gov.az")
	if len(contacts) != 1 {
		t.Fatalf("contacts %d", len(contacts))
	}
	if !strings.Contains(contacts[0].VCard, "John") {
		t.Fatalf("vcard %s", contacts[0].VCard)
	}
}

func TestGetItemAfterCreate(t *testing.T) {
	r := repo.NewMemory()
	_, _ = r.CreateMailbox("t1", "alice@mail.gov.az", "pw", 1<<20)
	h := &Handler{Repo: r, Cal: store.New()}
	create := `<?xml version="1.0"?><soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><CreateItem><Items><Message><Subject>Hi</Subject><Body>Text</Body></Message></Items></CreateItem></soap:Body></soap:Envelope>`
	req := httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", strings.NewReader(create))
	h.ServeHTTP(httptest.NewRecorder(), req)
	msgs, _ := r.ListMessages("alice@mail.gov.az")
	msg := msgs[0]

	get := `<?xml version="1.0"?><soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><GetItem><ItemIds><ItemId Id="` + formatMsgID(msg.ID) + `"/></ItemIds></GetItem></soap:Body></soap:Envelope>`
	req = httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", strings.NewReader(get))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Hi")) {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestSyncFolderItemsCalendar(t *testing.T) {
	cal := store.New()
	cal.Put("alice@mail.gov.az", "evt-ews-1", `BEGIN:VEVENT
UID:evt-ews-1
SUMMARY:Standup
DTSTART:20260707T090000Z
END:VEVENT`)
	r := repo.NewMemory()
	h := &Handler{Repo: r, Cal: cal}
	req := httptest.NewRequest(http.MethodPost, "/ews/Exchange.asmx", strings.NewReader(`<SyncFolderItems/>`))
	req.Header.Set("SOAPAction", "SyncFolderItems")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("sync %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("evt-ews-1")) {
		t.Fatalf("body %s", rec.Body.String())
	}
}
