// Package proxy — TCP/SSH proxy stub с записью команд в privilegedsession.
package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"era/services/platform/privilegedsession"
)

// SSHProxy — базовый TCP dial proxy к SSH-порту.
type SSHProxy struct {
	Sessions *privilegedsession.Store
	mu       sync.Mutex
	servers  map[string]net.Listener
}

func NewSSHProxy(sess *privilegedsession.Store) *SSHProxy {
	return &SSHProxy{Sessions: sess, servers: make(map[string]net.Listener)}
}

// Start поднимает локальный listener и проксирует на host:port; возвращает listen addr.
func (p *SSHProxy) Start(sessionID, host string, port int) (string, error) {
	if port <= 0 {
		port = 22
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	p.mu.Lock()
	p.servers[sessionID] = ln
	p.mu.Unlock()
	if p.Sessions != nil {
		p.Sessions.EnsureSession(sessionID, "pam-proxy", host)
	}
	go p.acceptLoop(sessionID, ln, net.JoinHostPort(host, strconv.Itoa(port)))
	return ln.Addr().String(), nil
}

func (p *SSHProxy) acceptLoop(sessionID string, ln net.Listener, target string) {
	for {
		client, err := ln.Accept()
		if err != nil {
			return
		}
		go p.relay(sessionID, client, target)
	}
}

func (p *SSHProxy) relay(sessionID string, client net.Conn, target string) {
	defer client.Close()
	remote, err := net.Dial("tcp", target)
	if err != nil {
		return
	}
	defer remote.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		p.copyAndLog(sessionID, client, remote)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(client, remote)
	}()
	wg.Wait()
}

func (p *SSHProxy) copyAndLog(sessionID string, src, dst net.Conn) {
	br := bufio.NewReader(src)
	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			cmd := strings.TrimSpace(line)
			if cmd != "" && p.Sessions != nil {
				p.Sessions.LogCommand(sessionID, cmd)
			}
			_, _ = dst.Write([]byte(line))
		}
		if err != nil {
			if err != io.EOF {
				rest, _ := io.ReadAll(br)
				if len(rest) > 0 {
					_, _ = dst.Write(rest)
				}
			}
			_, _ = io.Copy(dst, src)
			return
		}
	}
}

// Stop закрывает listener сессии.
func (p *SSHProxy) Stop(sessionID string) error {
	p.mu.Lock()
	ln := p.servers[sessionID]
	delete(p.servers, sessionID)
	p.mu.Unlock()
	if ln == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return ln.Close()
}
