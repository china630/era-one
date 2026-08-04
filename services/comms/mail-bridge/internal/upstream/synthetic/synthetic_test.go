package synthetic_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"era/services/comms/mail-bridge/internal/upstream/synthetic"
)

func TestSyntheticFindFolder(t *testing.T) {
	st, body, err := (synthetic.Backend{}).ProxyEWS(context.Background(), "FindFolder", nil, http.Header{})
	if err != nil || st != 200 || !strings.Contains(string(body), "INBOX") {
		t.Fatalf("st=%d err=%v body=%s", st, err, body)
	}
}
