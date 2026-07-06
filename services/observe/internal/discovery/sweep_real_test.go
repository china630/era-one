package discovery

import (
	"net"
	"testing"
)

func TestHostsInAllowlist(t *testing.T) {
	t.Setenv("ERA_DISCOVERY_ALLOWLIST", "127.0.0.1/32")
	ips, err := hostsInAllowlist("10.0.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 1 || ips[0] != "127.0.0.1" {
		t.Fatalf("allowlist filter: %+v", ips)
	}
}

func TestSweepRealLocalListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	t.Setenv("ERA_DISCOVERY_ALLOWLIST", host+"/32")
	t.Setenv("ERA_DISCOVERY_PROBE_PORTS", port)
	go func() {
		c, _ := ln.Accept()
		if c != nil {
			c.Close()
		}
	}()
	nodes, err := SweepReal("10.0.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range nodes {
		if n.IP == host {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s in sweep, got %+v", host, nodes)
	}
}

func TestParseAllowlist(t *testing.T) {
	list := parseAllowlist("10.0.0.0/24, 192.168.1.1")
	if len(list) != 2 {
		t.Fatalf("%v", list)
	}
}
