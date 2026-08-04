package notify

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// SMTPMailer отправляет служебные письма через SMTP.
type SMTPMailer struct {
	Addr    string
	Timeout time.Duration
	Dial    func(network, address string) (net.Conn, error)
}

func (m *SMTPMailer) Send(from string, to []string, subject, body string) error {
	if m == nil || m.Addr == "" {
		return fmt.Errorf("smtp mailer: empty addr")
	}
	dial := m.Dial
	if dial == nil {
		dial = net.Dial
	}
	timeout := m.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	type result struct {
		c   net.Conn
		err error
	}
	ch := make(chan result, 1)
	go func() {
		c, err := dial("tcp", m.Addr)
		ch <- result{c, err}
	}()
	var c net.Conn
	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		c = r.c
	case <-time.After(timeout):
		return fmt.Errorf("smtp mailer dial timeout")
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(timeout))
	rd := bufio.NewReader(c)
	wr := bufio.NewWriter(c)
	if _, err := readLine(rd); err != nil {
		return err
	}
	cmds := []string{"HELO era-mail-moderation", "MAIL FROM:<" + from + ">"}
	for _, t := range to {
		cmds = append(cmds, "RCPT TO:<"+t+">")
	}
	for _, cmd := range cmds {
		if err := writeLine(wr, cmd); err != nil {
			return err
		}
		if _, err := readLine(rd); err != nil {
			return err
		}
	}
	if err := writeLine(wr, "DATA"); err != nil {
		return err
	}
	if _, err := readLine(rd); err != nil {
		return err
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		from, strings.Join(to, ", "), subject, body)
	if _, err := io.WriteString(wr, msg); err != nil {
		return err
	}
	if err := writeLine(wr, "."); err != nil {
		return err
	}
	if _, err := readLine(rd); err != nil {
		return err
	}
	_ = writeLine(wr, "QUIT")
	_, _ = readLine(rd)
	return nil
}

func writeLine(w *bufio.Writer, s string) error {
	_, err := io.WriteString(w, s+"\r\n")
	if err != nil {
		return err
	}
	return w.Flush()
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

// MailerFromEnv — SMTP если addr задан, иначе Recorder.
func MailerFromEnv(addr string) Mailer {
	if strings.TrimSpace(addr) == "" {
		return &Recorder{}
	}
	return &SMTPMailer{Addr: strings.TrimSpace(addr)}
}
