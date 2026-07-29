package engine

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// SMTPUpstream доставляет письмо на внешний SMTP (IceWarp / mail-core).
type SMTPUpstream struct {
	Addr    string
	Timeout time.Duration
	Dial    func(network, address string) (net.Conn, error)
}

func (u *SMTPUpstream) Deliver(raw []byte, from string, to []string) error {
	if u == nil || u.Addr == "" {
		return fmt.Errorf("smtp upstream: empty addr")
	}
	dial := u.Dial
	if dial == nil {
		dial = net.Dial
	}
	timeout := u.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	type dialResult struct {
		c   net.Conn
		err error
	}
	ch := make(chan dialResult, 1)
	go func() {
		c, err := dial("tcp", u.Addr)
		ch <- dialResult{c, err}
	}()
	var c net.Conn
	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		c = r.c
	case <-time.After(timeout):
		return fmt.Errorf("smtp upstream dial timeout")
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(timeout))
	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)
	if _, err := readSMTP(r); err != nil {
		return err
	}
	cmds := []string{"HELO era-mail-moderation", "MAIL FROM:<" + from + ">"}
	for _, t := range to {
		cmds = append(cmds, "RCPT TO:<"+t+">")
	}
	cmds = append(cmds, "DATA")
	for _, cmd := range cmds {
		if err := writeSMTP(w, cmd); err != nil {
			return err
		}
		if _, err := readSMTP(r); err != nil {
			return err
		}
	}
	body := string(raw)
	if !strings.HasSuffix(body, "\r\n") {
		body += "\r\n"
	}
	if _, err := io.WriteString(w, body+".\r\n"); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if _, err := readSMTP(r); err != nil {
		return err
	}
	_ = writeSMTP(w, "QUIT")
	_, _ = readSMTP(r)
	return nil
}

func writeSMTP(w *bufio.Writer, s string) error {
	_, err := io.WriteString(w, s+"\r\n")
	if err != nil {
		return err
	}
	return w.Flush()
}

func readSMTP(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

// UpstreamFromEnv — SMTP если ERA_MM_UPSTREAM задан, иначе Memory.
func UpstreamFromEnv(addr string) Upstream {
	if strings.TrimSpace(addr) == "" {
		return &MemoryUpstream{}
	}
	return &SMTPUpstream{Addr: strings.TrimSpace(addr)}
}
