package notify_test

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"

	"era/services/comms/mail-moderation/internal/notify"
)

func TestSMTPMailer_Send(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	done := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			done <- err.Error()
			return
		}
		defer c.Close()
		r := bufio.NewReader(c)
		w := bufio.NewWriter(c)
		_, _ = io.WriteString(w, "220 ok\r\n")
		_ = w.Flush()
		var got strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				break
			}
			got.WriteString(line)
			u := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(u, "DATA"):
				_, _ = io.WriteString(w, "354\r\n")
				_ = w.Flush()
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						break
					}
					got.WriteString(l)
					if l == ".\r\n" || l == ".\n" {
						break
					}
				}
				_, _ = io.WriteString(w, "250 OK\r\n")
				_ = w.Flush()
			case u == "QUIT":
				_, _ = io.WriteString(w, "221\r\n")
				_ = w.Flush()
				done <- got.String()
				return
			default:
				_, _ = io.WriteString(w, "250 OK\r\n")
				_ = w.Flush()
			}
		}
		done <- got.String()
	}()

	m := &notify.SMTPMailer{Addr: ln.Addr().String()}
	if err := m.Send("mm@c.local", []string{"ivan@c.local"}, "subj", "body text"); err != nil {
		t.Fatal(err)
	}
	tr := <-done
	if !strings.Contains(tr, "MAIL FROM:<mm@c.local>") || !strings.Contains(tr, "Subject: subj") {
		t.Fatalf("%s", tr)
	}
}

func TestMailerFromEnv(t *testing.T) {
	if _, ok := notify.MailerFromEnv("").(*notify.Recorder); !ok {
		t.Fatal("empty")
	}
}
