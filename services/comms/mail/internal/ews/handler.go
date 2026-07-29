// Package ews — EWS SOAP subset for Outlook (Wave C-2 / R-GOV-1).
package ews

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"era/services/comms/calendar/store"
	"era/services/comms/mail/internal/repo"
)

// Handler serves POST /ews/Exchange.asmx.
type Handler struct {
	Repo    repo.Backend
	Cal     store.Backend
	Mailbox string
}

// ServeHTTP dispatches SOAP operations.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	raw := string(body)
	action := r.Header.Get("SOAPAction")
	if action == "" {
		action = detectAction(raw)
	}
	mailbox := h.Mailbox
	if mailbox == "" {
		mailbox = "alice@mail.gov.az"
	}
	switch {
	case strings.Contains(action, "FindFolder") || strings.Contains(raw, "FindFolder"):
		h.findFolder(w)
	case strings.Contains(action, "GetFolder") || strings.Contains(raw, "GetFolder"):
		h.getFolder(w)
	case strings.Contains(action, "CreateItem") || strings.Contains(raw, "CreateItem"):
		h.createItem(w, raw, mailbox)
	case strings.Contains(action, "UpdateItem") || strings.Contains(raw, "UpdateItem"):
		h.updateItem(w, raw, mailbox)
	case strings.Contains(action, "DeleteItem") || strings.Contains(raw, "DeleteItem"):
		h.deleteItem(w)
	case strings.Contains(action, "GetItem") || strings.Contains(raw, "GetItem"):
		h.getItem(w, raw, mailbox)
	case strings.Contains(action, "SyncFolderItems") || strings.Contains(raw, "SyncFolderItems"):
		h.syncFolderItems(w, mailbox)
	case strings.Contains(action, "SyncContacts") || strings.Contains(raw, "SyncContacts"):
		h.syncContacts(w, mailbox)
	case strings.Contains(action, "CreateContact") || strings.Contains(raw, "CreateContact"):
		h.createContact(w, raw, mailbox)
	case strings.Contains(action, "Subscribe") || strings.Contains(raw, "Subscribe"):
		h.subscribeStub(w)
	default:
		http.Error(w, "unsupported EWS operation", http.StatusNotImplemented)
	}
}

func detectAction(raw string) string {
	for _, op := range []string{
		"FindFolder", "GetFolder", "CreateItem", "UpdateItem", "DeleteItem",
		"GetItem", "SyncFolderItems", "SyncContacts", "CreateContact", "Subscribe",
	} {
		if strings.Contains(raw, op) {
			return op
		}
	}
	return ""
}

func (h *Handler) findFolder(w http.ResponseWriter) {
	writeSOAP(w, `<FindFolderResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages>
        <FindFolderResponseMessage ResponseClass="Success">
          <RootFolder><Folders><Folder><FolderId Id="inbox"/><DisplayName>Inbox</DisplayName></Folder></Folders></RootFolder>
        </FindFolderResponseMessage>
      </ResponseMessages>`)
}

func (h *Handler) getFolder(w http.ResponseWriter) {
	writeSOAP(w, `<GetFolderResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages>
        <GetFolderResponseMessage ResponseClass="Success">
          <Folders><Folder><FolderId Id="inbox"/><DisplayName>Inbox</DisplayName></Folder></Folders>
        </GetFolderResponseMessage>
      </ResponseMessages>`)
}

func (h *Handler) createItem(w http.ResponseWriter, raw, mailbox string) {
	subject := extractTag(raw, "Subject")
	body := extractTag(raw, "Body")
	if subject == "" {
		subject = "EWS Message"
	}
	msg, err := h.Repo.AddEWSMessage(mailbox, subject, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := formatMsgID(msg.ID)
	writeSOAP(w, `<CreateItemResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages>
        <CreateItemResponseMessage ResponseClass="Success">
          <Items><Message><ItemId Id="`+id+`"/></Message></Items>
        </CreateItemResponseMessage>
      </ResponseMessages>`)
}

func (h *Handler) updateItem(w http.ResponseWriter, raw, mailbox string) {
	id := extractAttr(raw, "ItemId", "Id")
	subject := extractTag(raw, "Subject")
	body := extractTag(raw, "Body")
	if id != "" {
		_, _ = h.Repo.AddEWSMessage(mailbox, subject, body)
	}
	writeSOAP(w, `<UpdateItemResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages><UpdateItemResponseMessage ResponseClass="Success"/></ResponseMessages>`)
}

func (h *Handler) deleteItem(w http.ResponseWriter) {
	writeSOAP(w, `<DeleteItemResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages><DeleteItemResponseMessage ResponseClass="Success"/></ResponseMessages>`)
}

func (h *Handler) getItem(w http.ResponseWriter, raw, mailbox string) {
	id := extractAttr(raw, "ItemId", "Id")
	msg, ok := h.Repo.GetMessageByID(mailbox, id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeSOAP(w, `<GetItemResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages>
        <GetItemResponseMessage ResponseClass="Success">
          <Items><Message><ItemId Id="`+formatMsgID(msg.ID)+`"/>
              <Subject>`+xmlEscape(msg.Subject)+`</Subject>
              <Body>`+xmlEscape(msg.Body)+`</Body>
            </Message></Items>
        </GetItemResponseMessage>
      </ResponseMessages>`)
}

func (h *Handler) syncFolderItems(w http.ResponseWriter, mailbox string) {
	msgs, _ := h.Repo.ListMessages(mailbox)
	events := h.Cal.List(mailbox)
	b := strings.Builder{}
	b.WriteString(`<SyncFolderItemsResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages><SyncFolderItemsResponseMessage ResponseClass="Success"><Changes>`)
	for _, msg := range msgs {
		b.WriteString(`<Create><Message><ItemId Id="`)
		b.WriteString(formatMsgID(msg.ID))
		b.WriteString(`"/><Subject>`)
		b.WriteString(xmlEscape(msg.Subject))
		b.WriteString(`</Subject></Message></Create>`)
	}
	for _, ev := range events {
		b.WriteString(`<Create><CalendarItem><ItemId Id="` + ev.UID + `"/><Subject>`)
		b.WriteString(xmlEscape(ev.UID))
		b.WriteString(`</Subject></CalendarItem></Create>`)
	}
	b.WriteString(`</Changes></SyncFolderItemsResponseMessage></ResponseMessages>`)
	writeSOAP(w, b.String())
}

func (h *Handler) syncContacts(w http.ResponseWriter, mailbox string) {
	contacts, _ := h.Repo.ListContacts(mailbox)
	b := strings.Builder{}
	b.WriteString(`<SyncContactsResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages><SyncContactsResponseMessage ResponseClass="Success"><Changes>`)
	for _, c := range contacts {
		b.WriteString(`<Create><Contact><ItemId Id="` + c.UID + `"/></Contact></Create>`)
	}
	b.WriteString(`</Changes></SyncContactsResponseMessage></ResponseMessages>`)
	writeSOAP(w, b.String())
}

func (h *Handler) createContact(w http.ResponseWriter, raw, mailbox string) {
	given := extractTag(raw, "GivenName")
	surname := extractTag(raw, "Surname")
	email := extractTag(raw, "EmailAddress")
	if email == "" {
		email = extractAttr(raw, "Entry", "Key")
		if strings.Contains(raw, "EmailAddress1") {
			email = extractEntryEmail(raw)
		}
	}
	uid := extractTag(raw, "ItemId")
	if uid == "" {
		uid = fmt.Sprintf("contact-%s-%s", given, surname)
		if given == "" && surname == "" {
			uid = fmt.Sprintf("contact-%d", time.Now().UnixNano())
		}
	}
	display := strings.TrimSpace(given + " " + surname)
	if display == "" {
		display = email
	}
	if display == "" {
		display = "Contact"
	}
	vcard := fmt.Sprintf("BEGIN:VCARD\r\nVERSION:3.0\r\nFN:%s\r\nN:%s;%s;;;\r\nEMAIL:%s\r\nEND:VCARD\r\n",
		display, surname, given, email)
	c, err := h.Repo.PutContact(mailbox, uid, vcard)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeSOAP(w, `<CreateContactResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages>
        <CreateContactResponseMessage ResponseClass="Success">
          <Contacts><Contact><ItemId Id="`+c.UID+`"/></Contact></Contacts>
        </CreateContactResponseMessage>
      </ResponseMessages>`)
}

func (h *Handler) subscribeStub(w http.ResponseWriter) {
	writeSOAP(w, `<SubscribeResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages><SubscribeResponseMessage ResponseClass="Success">
        <SubscriptionId>era-sub-1</SubscriptionId>
      </SubscribeResponseMessage></ResponseMessages>`)
}

func formatMsgID(id int64) string {
	return fmt.Sprintf("msg-%d", id)
}

func writeSOAP(w http.ResponseWriter, bodyInner string) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>` + bodyInner + `
  </soap:Body>
</soap:Envelope>`))
}

func extractTag(raw, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	i := strings.Index(raw, open)
	if i < 0 {
		return ""
	}
	j := strings.Index(raw[i:], close)
	if j < 0 {
		return ""
	}
	return raw[i+len(open) : i+j]
}

func extractAttr(raw, tag, attr string) string {
	idx := strings.Index(raw, "<"+tag)
	if idx < 0 {
		return ""
	}
	frag := raw[idx:]
	key := attr + `="`
	i := strings.Index(frag, key)
	if i < 0 {
		return ""
	}
	frag = frag[i+len(key):]
	j := strings.Index(frag, `"`)
	if j < 0 {
		return ""
	}
	return frag[:j]
}

func extractEntryEmail(raw string) string {
	idx := strings.Index(raw, "<Entry")
	for idx >= 0 {
		frag := raw[idx:]
		if strings.Contains(frag, "EmailAddress1") {
			close := strings.Index(frag, "</Entry>")
			if close < 0 {
				return ""
			}
			inner := frag[:close]
			gt := strings.LastIndex(inner, ">")
			if gt >= 0 {
				return strings.TrimSpace(inner[gt+1:])
			}
		}
		raw = raw[idx+1:]
		idx = strings.Index(raw, "<Entry")
	}
	return ""
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
