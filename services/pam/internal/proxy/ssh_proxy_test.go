package proxy

import (
	"net"
	"testing"
	"time"

	"era/services/platform/privilegedsession"
)

func TestSSHSessionTranscriptGolden(t *testing.T) {
	store := privilegedsession.NewStore()
	proxy := NewSSHProxy(store)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, _ := ln.Accept()
		if c != nil {
			_, _ = c.Write([]byte("SSH-2.0-test\r\n"))
			_ = c.Close()
		}
	}()
	addr, err := proxy.Start("sess-golden", "127.0.0.1", ln.Addr().(*net.TCPAddr).Port)
	if err != nil {
		t.Fatal(err)
	}
	client, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = client.Write([]byte("ls -la /tmp\n"))
	time.Sleep(50 * time.Millisecond)
	_ = client.Close()
	_ = proxy.Stop("sess-golden")
	rec, ok := store.End("sess-golden")
	if !ok || len(rec.Commands) == 0 {
		t.Fatal("expected command transcript")
	}
	if rec.Commands[0] != "ls -la /tmp" {
		t.Fatalf("transcript=%q", rec.Commands[0])
	}
}
