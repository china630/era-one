package proxy

import (
	"bufio"
	"io"
	"net"
	"strings"

	"era/services/platform/privilegedsession"
)

// SSHProxy — TCP dial proxy к SSH с построчным логом команд.
type SSHProxy struct {
	*tcpBridge
}

func NewSSHProxy(sess *privilegedsession.Store) *SSHProxy {
	p := &SSHProxy{tcpBridge: newBridge(sess, "pam-proxy")}
	p.relayFn = p.copyAndLogRelay
	return p
}

func (p *SSHProxy) Start(sessionID, host string, port int) (string, error) {
	if port <= 0 {
		port = 22
	}
	return p.start(sessionID, host, port)
}

func (p *SSHProxy) Stop(sessionID string) error { return p.stop(sessionID) }

func (p *SSHProxy) copyAndLogRelay(sessionID string, client net.Conn, target string) {
	defer client.Close()
	remote, err := net.Dial("tcp", target)
	if err != nil {
		return
	}
	defer remote.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(client, remote)
	}()
	br := bufio.NewReader(client)
	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			cmd := strings.TrimSpace(line)
			if cmd != "" && p.Sessions != nil {
				p.Sessions.LogCommand(sessionID, cmd)
			}
			_, _ = remote.Write([]byte(line))
		}
		if err != nil {
			if err != io.EOF {
				rest, _ := io.ReadAll(br)
				if len(rest) > 0 {
					_, _ = remote.Write(rest)
				}
			}
			_, _ = io.Copy(remote, client)
			<-done
			return
		}
	}
}
