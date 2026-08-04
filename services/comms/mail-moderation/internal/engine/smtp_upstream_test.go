package engine_test

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"

	"era/services/comms/mail-moderation/internal/engine"
)

func TestSMTPUpstream_Deliver(t *testing.T) {
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
		_, _ = io.WriteString(w, "220 mock\r\n")
		_ = w.Flush()
		var got strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				break
			}
			got.WriteString(line)
			upper := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(upper, "DATA"):
				_, _ = io.WriteString(w, "354 go\r\n")
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
			case upper == "QUIT":
				_, _ = io.WriteString(w, "221 bye\r\n")
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

	u := &engine.SMTPUpstream{Addr: ln.Addr().String()}
	if err := u.Deliver([]byte("Subject: t\r\n\r\nHi\r\n"), "a@b.c", []string{"x@y.z"}); err != nil {
		t.Fatal(err)
	}
	transcript := <-done
	if !strings.Contains(transcript, "MAIL FROM:<a@b.c>") || !strings.Contains(transcript, "RCPT TO:<x@y.z>") {
		t.Fatalf("transcript: %s", transcript)
	}
}

func TestUpstreamFromEnv(t *testing.T) {
	if _, ok := engine.UpstreamFromEnv("").(*engine.MemoryUpstream); !ok {
		t.Fatal("empty → memory")
	}
	if _, ok := engine.UpstreamFromEnv("127.0.0.1:25").(*engine.SMTPUpstream); !ok {
		t.Fatal("addr → smtp")
	}
}
