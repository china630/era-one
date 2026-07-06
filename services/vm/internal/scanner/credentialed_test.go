package scanner

import (
	"net"
	"testing"
	"time"

	"era/services/vm/internal/models"
)

func TestSSHCredentialedBanner(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Write([]byte("SSH-2.0-OpenSSH_8.9\r\n"))
	}()

	ex := &SSHCredentialedExecutor{Timeout: 2 * time.Second, Port: portOnly(ln.Addr().String())}
	tpl := &models.Template{ID: "ssh-banner", Info: models.Info{Name: "SSH exposed", Severity: "medium"}}
	findings, err := ex.Execute("127.0.0.1:"+portOnly(ln.Addr().String()), tpl)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].TemplateID != "ssh-banner" {
		t.Fatalf("findings=%#v", findings)
	}
}

func portOnly(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "22"
	}
	return port
}
