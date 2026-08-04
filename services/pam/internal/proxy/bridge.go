package proxy

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"era/services/platform/privilegedsession"
)

type tcpBridge struct {
	Sessions *privilegedsession.Store
	mu       sync.Mutex
	servers  map[string]net.Listener
	role     string
	// if set, used instead of binary relay (SSH line logger)
	relayFn func(sessionID string, client net.Conn, target string)
}

func newBridge(sess *privilegedsession.Store, role string) *tcpBridge {
	return &tcpBridge{Sessions: sess, servers: make(map[string]net.Listener), role: role}
}

func (p *tcpBridge) start(sessionID, host string, port int) (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	p.mu.Lock()
	p.servers[sessionID] = ln
	p.mu.Unlock()
	if p.Sessions != nil {
		p.Sessions.EnsureSession(sessionID, p.role, host)
	}
	go p.acceptLoop(sessionID, ln, net.JoinHostPort(host, strconv.Itoa(port)))
	return ln.Addr().String(), nil
}

func (p *tcpBridge) acceptLoop(sessionID string, ln net.Listener, target string) {
	for {
		client, err := ln.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			if p.relayFn != nil {
				p.relayFn(sessionID, c, target)
				return
			}
			p.binaryRelay(c, target)
		}(client)
	}
}

func (p *tcpBridge) binaryRelay(client net.Conn, target string) {
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
		_, _ = io.Copy(remote, client)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(client, remote)
	}()
	wg.Wait()
}

func (p *tcpBridge) stop(sessionID string) error {
	p.mu.Lock()
	ln := p.servers[sessionID]
	delete(p.servers, sessionID)
	p.mu.Unlock()
	if ln == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return ln.Close()
}
