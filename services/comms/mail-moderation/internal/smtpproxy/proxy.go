// Package smtpproxy — упрощённый SMTP submission edge (AC-MM-2,9).
package smtpproxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"era/services/comms/mail-moderation/internal/engine"
	"era/services/comms/mail-moderation/internal/policy"
)

// Server — минимальный SMTP (HELO/MAIL/RCPT/DATA/QUIT), без полного RFC compliance.
type Server struct {
	Engine *engine.Engine
	ln     net.Listener
	wg     sync.WaitGroup
}

func (s *Server) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.ln = ln
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			s.wg.Add(1)
			go func(conn net.Conn) {
				defer s.wg.Done()
				s.handle(conn)
			}(c)
		}
	}()
	return nil
}

func (s *Server) Addr() net.Addr {
	if s.ln == nil {
		return nil
	}
	return s.ln.Addr()
}

func (s *Server) Close() error {
	if s.ln != nil {
		_ = s.ln.Close()
	}
	s.wg.Wait()
	return nil
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	writeLine(w, "220 era-mail-moderation ESMTP")
	var from string
	var to []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "HELO") || strings.HasPrefix(upper, "EHLO"):
			writeLine(w, "250 OK")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			from = extractAngle(line[10:])
			to = nil
			writeLine(w, "250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			to = append(to, extractAngle(line[8:]))
			writeLine(w, "250 OK")
		case upper == "DATA":
			writeLine(w, "354 End data with <CR><LF>.<CR><LF>")
			var raw strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" || l == ".\n" {
					break
				}
				raw.WriteString(l)
			}
			dec, holdID, err := s.Engine.ProcessRaw([]byte(raw.String()), from, to)
			if err != nil {
				writeLine(w, "451 "+err.Error())
				continue
			}
			switch dec {
			case policy.DecisionHold:
				writeLine(w, fmt.Sprintf("250 Queued for moderation id=%s", holdID))
			default:
				writeLine(w, "250 OK queued")
			}
		case upper == "QUIT":
			writeLine(w, "221 Bye")
			return
		case upper == "RSET":
			from, to = "", nil
			writeLine(w, "250 OK")
		case upper == "NOOP":
			writeLine(w, "250 OK")
		default:
			writeLine(w, "502 Command not implemented")
		}
	}
}

func writeLine(w *bufio.Writer, s string) {
	_, _ = io.WriteString(w, s+"\r\n")
	_ = w.Flush()
}

func extractAngle(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "<"); i >= 0 {
		if j := strings.Index(s[i:], ">"); j > 0 {
			return s[i+1 : i+j]
		}
	}
	return strings.Trim(s, "<>")
}

// Submit — тестовый клиент к proxy.
func Submit(addr, from string, to []string, raw string) (string, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return "", err
	}
	defer c.Close()
	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)
	if _, err := readResp(r); err != nil {
		return "", err
	}
	cmds := []string{"HELO test", "MAIL FROM:<" + from + ">"}
	for _, t := range to {
		cmds = append(cmds, "RCPT TO:<"+t+">")
	}
	cmds = append(cmds, "DATA")
	var last string
	for _, cmd := range cmds {
		_, _ = io.WriteString(w, cmd+"\r\n")
		_ = w.Flush()
		last, err = readResp(r)
		if err != nil {
			return last, err
		}
	}
	_, _ = io.WriteString(w, raw)
	if !strings.HasSuffix(raw, "\r\n") {
		_, _ = io.WriteString(w, "\r\n")
	}
	_, _ = io.WriteString(w, ".\r\n")
	_ = w.Flush()
	last, err = readResp(r)
	if err != nil {
		return last, err
	}
	_, _ = io.WriteString(w, "QUIT\r\n")
	_ = w.Flush()
	_, _ = readResp(r)
	return last, nil
}

func readResp(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}
