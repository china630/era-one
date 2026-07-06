package ndr

import (
	"testing"
	"time"
)

func TestBeaconingRegularInterval(t *testing.T) {
	e := New(30 * time.Minute)
	at := time.Now().UTC()
	src, dst := "10.1.1.5", "203.0.113.9"
	for i := 0; i < 6; i++ {
		ts := at.Add(time.Duration(i*60) * time.Second)
		e.ObserveBeacon(src, dst, ts)
	}
	ok, rule := e.Beaconing(src, dst)
	if !ok || rule != "era-ndr-t1071-beaconing" {
		t.Fatalf("beaconing ok=%v rule=%s", ok, rule)
	}
}

func TestDNSTunnelLongLabels(t *testing.T) {
	e := New(30 * time.Minute)
	at := time.Now().UTC()
	src := "10.1.1.8"
	for i := 0; i < 10; i++ {
		q := "abcdefghijklmnopqrstuvwxyz0123456789abcdef.example.com"
		e.ObserveDNS(src, q, at.Add(time.Duration(i)*time.Second))
	}
	ok, rule := e.DNSTunnel(src)
	if !ok || rule != "era-ndr-t1071-004-dns-tunnel" {
		t.Fatalf("dns tunnel ok=%v rule=%s", ok, rule)
	}
}

func TestT1021LateralMovement(t *testing.T) {
	e := New(15 * time.Minute)
	at := time.Now().UTC()
	src := "10.0.0.99"
	e.Observe(src, "10.0.0.10", 445, at)
	e.Observe(src, "10.0.0.11", 3389, at.Add(time.Second))
	e.Observe(src, "10.0.0.12", 5985, at.Add(2*time.Second))
	ok, rule := e.LateralMovement(src)
	if !ok || rule != "era-ndr-t1021-lateral-movement" {
		t.Fatalf("expected T1021 detection, ok=%v rule=%s", ok, rule)
	}
}
