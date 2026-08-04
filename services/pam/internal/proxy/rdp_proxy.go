package proxy

import "era/services/platform/privilegedsession"

// RDPProxy — TCP dial proxy к RDP (3389); бинарный relay.
// Graphical recording / credential inject — Phase 2 (security-review).
type RDPProxy struct {
	*tcpBridge
}

func NewRDPProxy(sess *privilegedsession.Store) *RDPProxy {
	return &RDPProxy{tcpBridge: newBridge(sess, "pam-rdp")}
}

func (p *RDPProxy) Start(sessionID, host string, port int) (string, error) {
	if port <= 0 {
		port = 3389
	}
	return p.start(sessionID, host, port)
}

func (p *RDPProxy) Stop(sessionID string) error { return p.stop(sessionID) }
