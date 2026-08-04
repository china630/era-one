package doh

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"era/services/resolve/internal/atlas"
	"era/services/resolve/internal/dnsx"
	"era/services/resolve/internal/guard"
	"era/services/resolve/internal/policy"
	"era/services/resolve/internal/trace"
)

func TestDoHPostGoldenVerdict(t *testing.T) {
	pol := policy.NewStore()
	atl := atlas.New()
	eng := guard.New(pol, atl)
	tr := trace.New(8, nil)
	dns := &dnsx.Server{Guard: eng, Trace: tr}
	h := &Handler{DNS: dns, Enabled: func() bool { return true }}

	wire := encodeQuery("lab.malware.test", 1)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dns-query", bytes.NewReader(wire))
	req.Header.Set("Content-Type", "application/dns-message")
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if dnsx.Rcode(rec.Body.Bytes()) != 3 {
		t.Fatalf("want NXDOMAIN rcode=3 got %d", dnsx.Rcode(rec.Body.Bytes()))
	}

	b64 := base64.RawURLEncoding.EncodeToString(wire)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dns-query?dns="+b64, nil))
	if rec.Code != 200 || dnsx.Rcode(rec.Body.Bytes()) != 3 {
		t.Fatalf("GET doh %d rcode=%d", rec.Code, dnsx.Rcode(rec.Body.Bytes()))
	}
}

func TestDoHDisabled(t *testing.T) {
	h := &Handler{DNS: &dnsx.Server{Guard: guard.New(policy.NewStore(), atlas.New())}, Enabled: func() bool { return false }}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/dns-query", bytes.NewReader(encodeQuery("x.com", 1))))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("%d", rec.Code)
	}
}

func encodeQuery(qname string, qtype uint16) []byte {
	hdr := make([]byte, 12)
	binary.BigEndian.PutUint16(hdr[0:2], 0x1234)
	binary.BigEndian.PutUint16(hdr[2:4], 0x0100)
	binary.BigEndian.PutUint16(hdr[4:6], 1)
	var name []byte
	for _, lab := range strings.Split(strings.TrimSuffix(qname, "."), ".") {
		name = append(name, byte(len(lab)))
		name = append(name, []byte(lab)...)
	}
	name = append(name, 0)
	qt := make([]byte, 4)
	binary.BigEndian.PutUint16(qt[0:2], qtype)
	binary.BigEndian.PutUint16(qt[2:4], 1)
	return append(append(hdr, name...), qt...)
}
