package ndr

import (
	"testing"
	"time"
)

func TestNDRGoldenScenarios(t *testing.T) {
	e := New(30 * time.Minute)
	at := time.Now().UTC()
	src, dst := "10.1.1.5", "203.0.113.9"
	for i := 0; i < 6; i++ {
		e.ObserveBeacon(src, dst, at.Add(time.Duration(i*60)*time.Second))
	}
	if ok, _ := e.Beaconing(src, dst); !ok {
		t.Fatal("beaconing golden")
	}
	for i := 0; i < 10; i++ {
		e.ObserveDNS("10.1.1.8", "abcdefghijklmnopqrstuvwxyz0123456789abcdef.example.com", at.Add(time.Duration(i)*time.Second))
	}
	if ok, _ := e.DNSTunnel("10.1.1.8"); !ok {
		t.Fatal("dns tunnel golden")
	}
	e.ObserveExfil("10.0.0.5", 12_000_000, "e7d705a3286e19ea42f587b344ee6865", at)
	if ok, _ := e.JA3Fingerprint("10.0.0.5"); !ok {
		t.Fatal("ja3 golden")
	}
}
