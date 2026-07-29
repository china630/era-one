package upstream

import (
	"context"
	"net/http"
)

// Backend proxies EWS operations to an upstream mail server.
type Backend interface {
	Name() string
	ProxyEWS(ctx context.Context, soapAction string, body []byte, headers http.Header) (status int, respBody []byte, err error)
}
