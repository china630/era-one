package correlator

import (
	"testing"
	"time"
)

func TestAPTChain(t *testing.T) {
	e := New(10 * time.Minute)
	at := time.Now().UTC()
	node := "node-apt-1"

	e.Observe(node, "process", "e1", `{"image_path":"powershell.exe","command_line":"powershell -enc x"}`, at)
	e.Observe(node, "network", "e2", `{"dst_ip":"192.168.1.10","dst_port":445}`, at.Add(time.Second))
	e.Observe(node, "auth", "e3", `{"user":"bob","success":false}`, at.Add(2*time.Second))

	ok, rule := e.APTChain(node)
	if !ok || rule != "era-apt-lateral-movement" {
		t.Fatalf("expected APT chain, got ok=%v rule=%s", ok, rule)
	}
}

func TestObserveNetworkEndpoint(t *testing.T) {
	e := New(10 * time.Minute)
	at := time.Now().UTC()
	node := "node-obs-1"
	e.Observe(node, "network", "n1", `{"summary":"PRTG high egress on uplink","source":"prtg"}`, at)
	e.Observe(node, "process", "p1", `{"image_path":"powershell.exe","command_line":"suspicious malware dropper"}`, at.Add(time.Second))
	ok, rule := e.ObserveNetworkEndpoint(node)
	if !ok || rule != "era-observe-network-endpoint" {
		t.Fatalf("expected observe chain, got ok=%v rule=%s", ok, rule)
	}
}