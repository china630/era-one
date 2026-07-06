package ndr

import (
	"testing"
	"time"
)

func TestDataExfilThreshold(t *testing.T) {
	e := New(30 * time.Minute)
	at := time.Now().UTC()
	src := "10.2.0.5"
	chunk := uint64(12 * 1024 * 1024)
	for i := 0; i < 5; i++ {
		e.ObserveExfil(src, chunk, "", at.Add(time.Duration(i)*time.Second))
	}
	ok, rule := e.DataExfil(src)
	if !ok || rule != "era-ndr-t1048-data-exfil" {
		t.Fatalf("exfil ok=%v rule=%s", ok, rule)
	}
}

func TestJA3FingerprintMalicious(t *testing.T) {
	e := New(30 * time.Minute)
	at := time.Now().UTC()
	src := "10.2.0.9"
	ja3 := "e7d705a3286e19ea42f587b344ee6865"
	e.ObserveExfil(src, 1024, ja3, at)
	ok, rule := e.JA3Fingerprint(src)
	if !ok || rule != "era-ndr-ja3-cobalt-strike" {
		t.Fatalf("ja3 ok=%v rule=%s", ok, rule)
	}
}

func TestParseNetworkExfil(t *testing.T) {
	payload := `{"src_ip":"10.1.1.1","bytes_out":4096,"ja3":"abc123"}`
	bytes, ja3 := ParseNetworkExfil(payload)
	if bytes != 4096 || ja3 != "abc123" {
		t.Fatalf("bytes=%d ja3=%s", bytes, ja3)
	}
}
