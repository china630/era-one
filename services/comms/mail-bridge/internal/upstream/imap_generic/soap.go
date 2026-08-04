package imap_generic

import (
	"fmt"
	"strings"
	"time"

	"era/services/comms/internal/imapclient"
)

func soapEnvelope(inner string) []byte {
	return []byte(`<?xml version="1.0" encoding="utf-8"?><soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body>` + inner + `</soap:Body></soap:Envelope>`)
}

func findFolderResponse(folders []string) []byte {
	var b strings.Builder
	b.WriteString(`<FindFolderResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages"><ResponseMessages><FindFolderResponseMessage ResponseClass="Success"><RootFolder><Folders>`)
	for _, f := range folders {
		id := folderID(f)
		b.WriteString(fmt.Sprintf(`<Folder><FolderId Id="%s"/><DisplayName>%s</DisplayName></Folder>`, id, xmlEscape(f)))
	}
	b.WriteString(`</Folders></RootFolder></FindFolderResponseMessage></ResponseMessages></FindFolderResponse>`)
	return soapEnvelope(b.String())
}

func syncFolderItemsResponse(msgs []imapclient.Message) []byte {
	var b strings.Builder
	b.WriteString(`<SyncFolderItemsResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages"><ResponseMessages><SyncFolderItemsResponseMessage ResponseClass="Success"><Changes>`)
	for _, m := range msgs {
		subj := m.Subject
		if subj == "" {
			subj = "message"
		}
		b.WriteString(fmt.Sprintf(`<Create><ItemId Id="%d"/><Subject>%s</Subject></Create>`, m.UID, xmlEscape(subj)))
	}
	b.WriteString(`</Changes></SyncFolderItemsResponseMessage></ResponseMessages></SyncFolderItemsResponse>`)
	return soapEnvelope(b.String())
}

func createItemResponse(uid uint32) []byte {
	inner := fmt.Sprintf(`<CreateItemResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages"><ResponseMessages><CreateItemResponseMessage ResponseClass="Success"><Items><Message><ItemId Id="%d"/></Message></Items></CreateItemResponseMessage></ResponseMessages></CreateItemResponse>`, uid)
	return soapEnvelope(inner)
}

func buildRFC822(subject, body, from string) []byte {
	if from == "" {
		from = "user@local"
	}
	now := time.Now().UTC().Format(time.RFC1123Z)
	return []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\n\r\n%s\r\n", from, from, subject, now, body))
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

func folderID(name string) string {
	switch strings.ToUpper(name) {
	case "INBOX":
		return "inbox"
	case "SENT", "SENT ITEMS", "SENT ITEMS/":
		return "sent"
	default:
		return strings.ReplaceAll(strings.ToLower(name), " ", "-")
	}
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	return strings.ReplaceAll(s, ">", "&gt;")
}
