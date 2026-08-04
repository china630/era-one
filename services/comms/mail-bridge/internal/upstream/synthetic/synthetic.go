package synthetic

import (
	"context"
	"net/http"
	"strings"
)

// Backend — lab SOAP echo (Summer S2-B); EWS ≠ 502 without real Exchange.
type Backend struct{}

func (Backend) Name() string { return "synthetic" }

func (Backend) ProxyEWS(_ context.Context, soapAction string, body []byte, _ http.Header) (int, []byte, error) {
	action := soapAction
	raw := string(body)
	inner := `<FindFolderResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages><FindFolderResponseMessage ResponseClass="Success">
        <RootFolder><Folders><Folder><DisplayName>INBOX</DisplayName></Folder></Folders></RootFolder>
      </FindFolderResponseMessage></ResponseMessages>`
	if strings.Contains(action, "CreateItem") || strings.Contains(raw, "CreateItem") {
		inner = `<CreateItemResponse xmlns="http://schemas.microsoft.com/exchange/services/2006/messages">
      <ResponseMessages><CreateItemResponseMessage ResponseClass="Success">
        <Items><Message><ItemId Id="synth-1"/></Message></Items>
      </CreateItemResponseMessage></ResponseMessages>`
	}
	out := []byte(`<?xml version="1.0"?><soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body>` + inner + `</soap:Body></soap:Envelope>`)
	return http.StatusOK, out, nil
}
