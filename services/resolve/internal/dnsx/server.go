// Package dnsx — minimal DNS UDP/TCP serve for Guard (RFC 1035 subset).
package dnsx

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"era/services/resolve/internal/guard"
	"era/services/resolve/internal/policy"
	"era/services/resolve/internal/trace"
)

// Server answers DNS queries via Guard.
type Server struct {
	Guard  *guard.Engine
	Trace  *trace.Buffer
	Addr   string
	udp    *net.UDPConn
}

func (s *Server) ListenAndServe() error {
	ua, err := net.ResolveUDPAddr("udp", s.Addr)
	if err != nil {
		return err
	}
	c, err := net.ListenUDP("udp", ua)
	if err != nil {
		return err
	}
	s.udp = c
	return s.serveLoop()
}

func (s *Server) LocalAddr() string {
	if s.udp == nil {
		return ""
	}
	return s.udp.LocalAddr().String()
}

func (s *Server) serveLoop() error {
	buf := make([]byte, 2048)
	for {
		n, addr, err := s.udp.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		resp, err := s.handle(buf[:n])
		if err != nil || len(resp) == 0 {
			continue
		}
		_, _ = s.udp.WriteToUDP(resp, addr)
	}
}

// Listen binds without serving (tests).
func (s *Server) Listen() error {
	ua, err := net.ResolveUDPAddr("udp", s.Addr)
	if err != nil {
		return err
	}
	c, err := net.ListenUDP("udp", ua)
	if err != nil {
		return err
	}
	s.udp = c
	return nil
}

func (s *Server) Close() error {
	if s.udp != nil {
		return s.udp.Close()
	}
	return nil
}

func (s *Server) handle(msg []byte) ([]byte, error) {
	return s.HandleMessage(msg)
}

// HandleMessage evaluates a raw DNS query with Guard (UDP + DoH).
func (s *Server) HandleMessage(msg []byte) ([]byte, error) {
	qname, qtype, id, err := parseQuery(msg)
	if err != nil {
		return nil, err
	}
	v := s.Guard.Decide(qname, qtypeName(qtype))
	if s.Trace != nil {
		s.Trace.Record(v)
	}
	return buildResponse(msg, id, qname, qtype, v)
}

func qtypeName(t uint16) string {
	switch t {
	case 1:
		return "A"
	case 28:
		return "AAAA"
	case 16:
		return "TXT"
	default:
		return fmt.Sprintf("%d", t)
	}
}

func parseQuery(msg []byte) (qname string, qtype uint16, id uint16, err error) {
	if len(msg) < 12 {
		return "", 0, 0, fmt.Errorf("short")
	}
	id = binary.BigEndian.Uint16(msg[0:2])
	off := 12
	var labels []string
	for {
		if off >= len(msg) {
			return "", 0, 0, fmt.Errorf("bad name")
		}
		l := int(msg[off])
		off++
		if l == 0 {
			break
		}
		if off+l > len(msg) {
			return "", 0, 0, fmt.Errorf("label overflow")
		}
		labels = append(labels, string(msg[off:off+l]))
		off += l
	}
	if off+4 > len(msg) {
		return "", 0, 0, fmt.Errorf("no qtype")
	}
	qtype = binary.BigEndian.Uint16(msg[off : off+2])
	return strings.Join(labels, "."), qtype, id, nil
}

func buildResponse(req []byte, id uint16, qname string, qtype uint16, v guard.Verdict) ([]byte, error) {
	// Copy question section from request header+question
	if len(req) < 12 {
		return nil, fmt.Errorf("short")
	}
	out := make([]byte, 0, 512)
	hdr := make([]byte, 12)
	copy(hdr, req[:12])
	binary.BigEndian.PutUint16(hdr[0:2], id)
	// QR=1, RD copy, RA=1
	flags := binary.BigEndian.Uint16(hdr[2:4])
	flags |= 0x8000 // QR
	flags |= 0x0080 // RA
	rcode := uint16(0)
	ancount := uint16(0)
	var answer []byte
	switch v.Action {
	case policy.ActionNXDomain:
		rcode = 3 // NXDOMAIN
	case policy.ActionSinkhole:
		if qtype == 1 && v.Sinkhole != "" {
			ip := net.ParseIP(v.Sinkhole).To4()
			if ip != nil {
				ancount = 1
				answer = encodeA(qname, ip)
			}
		}
	case policy.ActionAllow:
		// No recursive resolve in MVP — empty answer (NOERROR)
	}
	flags = (flags &^ 0x000F) | rcode
	binary.BigEndian.PutUint16(hdr[2:4], flags)
	binary.BigEndian.PutUint16(hdr[4:6], 1) // QDCOUNT
	binary.BigEndian.PutUint16(hdr[6:8], ancount)
	binary.BigEndian.PutUint16(hdr[8:10], 0)
	binary.BigEndian.PutUint16(hdr[10:12], 0)
	out = append(out, hdr...)
	// question from request
	qend := 12
	for qend < len(req) {
		if req[qend] == 0 {
			qend += 5 // 0 + type + class
			break
		}
		qend += 1 + int(req[qend])
	}
	if qend > len(req) {
		qend = len(req)
	}
	out = append(out, req[12:qend]...)
	out = append(out, answer...)
	return out, nil
}

func encodeA(qname string, ip net.IP) []byte {
	var b []byte
	for _, lab := range strings.Split(strings.TrimSuffix(qname, "."), ".") {
		b = append(b, byte(len(lab)))
		b = append(b, []byte(lab)...)
	}
	b = append(b, 0)
	rr := make([]byte, 10)
	binary.BigEndian.PutUint16(rr[0:2], 1)  // TYPE A
	binary.BigEndian.PutUint16(rr[2:4], 1)  // CLASS IN
	binary.BigEndian.PutUint32(rr[4:8], 60) // TTL
	binary.BigEndian.PutUint16(rr[8:10], 4) // RDLENGTH
	b = append(b, rr...)
	b = append(b, ip...)
	return b
}

// QueryUDP sends a query and returns raw response (tests).
func QueryUDP(addr, qname string, qtype uint16) ([]byte, error) {
	c, err := net.Dial("udp", addr)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	msg := encodeQuery(qname, qtype)
	if _, err := c.Write(msg); err != nil {
		return nil, err
	}
	buf := make([]byte, 2048)
	n, err := c.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func encodeQuery(qname string, qtype uint16) []byte {
	hdr := make([]byte, 12)
	binary.BigEndian.PutUint16(hdr[0:2], 0x1234)
	binary.BigEndian.PutUint16(hdr[2:4], 0x0100) // RD
	binary.BigEndian.PutUint16(hdr[4:6], 1)
	var name []byte
	for _, lab := range strings.Split(strings.TrimSuffix(qname, "."), ".") {
		name = append(name, byte(len(lab)))
		name = append(name, []byte(lab)...)
	}
	name = append(name, 0)
	qt := make([]byte, 4)
	binary.BigEndian.PutUint16(qt[0:2], qtype)
	binary.BigEndian.PutUint16(qt[2:4], 1)
	return append(append(hdr, name...), qt...)
}

// Rcode extracts response code from DNS message.
func Rcode(msg []byte) int {
	if len(msg) < 4 {
		return -1
	}
	return int(binary.BigEndian.Uint16(msg[2:4]) & 0x000F)
}

// HasAnswerA returns true if an A RR is present.
func HasAnswerA(msg []byte) bool {
	if len(msg) < 12 {
		return false
	}
	return binary.BigEndian.Uint16(msg[6:8]) > 0
}
