// Package imapclient — shared RFC 3501 subset for migration and mail-bridge.
package imapclient

import (
	"bufio"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Config — IMAP connection parameters.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      bool
}

// Message — fetched mailbox message.
type Message struct {
	UID     uint32
	Folder  string
	Raw     []byte
	Hash    string
	Subject string
	Seen    bool
}

// Client wraps a minimal IMAP session.
type Client struct {
	cfg    Config
	conn   net.Conn
	reader *bufio.Reader
	tag    int
}

// Dial connects and authenticates.
func Dial(cfg Config) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	var (
		conn net.Conn
		err  error
	)
	if cfg.TLS {
		conn, err = tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec // lab testbed
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	cl := &Client{cfg: cfg, conn: conn, reader: bufio.NewReader(conn)}
	if _, err := cl.readLine(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := cl.login(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return cl, nil
}

// Close closes connection.
func (cl *Client) Close() error {
	if cl.conn == nil {
		return nil
	}
	_, _ = cl.cmd("LOGOUT")
	return cl.conn.Close()
}

func (cl *Client) login() error {
	_, err := cl.cmd(`LOGIN "%s" "%s"`, escape(cl.cfg.Username), escape(cl.cfg.Password))
	return err
}

// ListMailboxes returns mailbox names.
func (cl *Client) ListMailboxes() ([]string, error) {
	mboxes, err := cl.ListMailboxesDetailed()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(mboxes))
	for _, mb := range mboxes {
		out = append(out, mb.Name)
	}
	return out, nil
}

// ListMailboxesDetailed returns mailboxes with LIST attributes (G4 special-use).
func (cl *Client) ListMailboxesDetailed() ([]Mailbox, error) {
	lines, err := cl.cmdMultiline(`LIST "" "*"`)
	if err != nil {
		return nil, err
	}
	var out []Mailbox
	for _, line := range lines {
		if !strings.HasPrefix(line, "* LIST") {
			continue
		}
		mb := parseListLine(line)
		if mb.Name != "" {
			out = append(out, mb)
		}
	}
	return out, nil
}

func parseListLine(line string) Mailbox {
	attrs := parseListAttributes(line)
	parts := strings.Split(line, `"`)
	if len(parts) < 2 {
		return Mailbox{Attributes: attrs}
	}
	return Mailbox{Name: parts[len(parts)-2], Attributes: attrs}
}

func parseListAttributes(line string) []string {
	i := strings.Index(line, "(")
	if i < 0 {
		return nil
	}
	j := strings.Index(line[i:], ")")
	if j < 0 {
		return nil
	}
	inner := strings.TrimSpace(line[i+1 : i+j])
	if inner == "" {
		return nil
	}
	return strings.Fields(inner)
}

// Append stores raw RFC822 in folder. When seen is true, message is stored with (\Seen).
func (cl *Client) Append(folder string, raw []byte, seen bool) error {
	if _, err := cl.cmd(`SELECT "%s"`, escape(folder)); err != nil {
		if _, err2 := cl.cmd(`CREATE "%s"`, escape(folder)); err2 != nil {
			return err
		}
		if _, err = cl.cmd(`SELECT "%s"`, escape(folder)); err != nil {
			return err
		}
	}
	tag := cl.nextTag()
	var err error
	if seen {
		_, err = fmt.Fprintf(cl.conn, "%s APPEND \"%s\" (\\Seen) { %d}\r\n", tag, escape(folder), len(raw))
	} else {
		_, err = fmt.Fprintf(cl.conn, "%s APPEND \"%s\" { %d}\r\n", tag, escape(folder), len(raw))
	}
	if err != nil {
		return err
	}
	cont, err := cl.readLine()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(cont, "+") {
		return fmt.Errorf("append literal: %s", cont)
	}
	if _, err := cl.conn.Write(raw); err != nil {
		return err
	}
	if _, err := cl.conn.Write([]byte("\r\n")); err != nil {
		return err
	}
	line, err := cl.readLine()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, tag+" OK") {
		return fmt.Errorf("append failed: %s", line)
	}
	return nil
}

// HashRaw returns stable SHA256 hex for golden tests.
func HashRaw(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// WaitReady sleeps briefly for async test servers.
func WaitReady() {
	time.Sleep(100 * time.Millisecond)
}

func (cl *Client) cmd(format string, args ...any) (string, error) {
	tag := cl.nextTag()
	line := fmt.Sprintf("%s "+format, append([]any{tag}, args...)...)
	if _, err := fmt.Fprintf(cl.conn, "%s\r\n", line); err != nil {
		return "", err
	}
	for {
		resp, err := cl.readLine()
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(resp, tag+" OK") {
			return resp, nil
		}
		if strings.HasPrefix(resp, tag+" NO") || strings.HasPrefix(resp, tag+" BAD") {
			return resp, fmt.Errorf("imap: %s", resp)
		}
	}
}

func (cl *Client) cmdMultiline(format string, args ...any) ([]string, error) {
	tag := cl.nextTag()
	line := fmt.Sprintf("%s "+format, append([]any{tag}, args...)...)
	if _, err := fmt.Fprintf(cl.conn, "%s\r\n", line); err != nil {
		return nil, err
	}
	var lines []string
	for {
		resp, err := cl.readLine()
		if err != nil {
			return nil, err
		}
		lines = append(lines, resp)
		if strings.HasPrefix(resp, tag+" OK") {
			break
		}
	}
	return lines, nil
}

func (cl *Client) nextTag() string {
	cl.tag++
	return fmt.Sprintf("a%d", cl.tag)
}

func (cl *Client) readLine() (string, error) {
	line, err := cl.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func escape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func parseSubject(raw []byte) string {
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.ToLower(line), "subject:") {
			return strings.TrimSpace(line[8:])
		}
	}
	return ""
}

// Ensure io usage for append path compile
var _ = io.EOF
