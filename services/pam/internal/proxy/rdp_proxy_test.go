package proxy

import (
	"io"
	"net"
	"testing"

	"era/services/platform/privilegedsession"
)

func TestRDPProxyTCPRelay(t *testing.T) {
	store := privilegedsession.NewStore()
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	go func() {
		c, err := backend.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 4)
		_, _ = io.ReadFull(c, buf)
		_, _ = c.Write([]byte("PONG"))
	}()

	p := NewRDPProxy(store)
	port := backend.Addr().(*net.TCPAddr).Port
	addr, err := p.Start("rdp-sess", "127.0.0.1", port)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Stop("rdp-sess") }()

	client, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_, _ = client.Write([]byte("PING"))
	buf := make([]byte, 4)
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "PONG" {
		t.Fatalf("got %q", buf)
	}
}
