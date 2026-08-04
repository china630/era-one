package archive

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"era/services/comms/internal/imapclient"
)

// ImportMBOX reads mboxrd format into messages.
func ImportMBOX(path string) ([]imapclient.Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseMBOX(f)
}

// ParseMBOX parses mboxrd from reader.
func ParseMBOX(r io.Reader) ([]imapclient.Message, error) {
	sc := bufio.NewScanner(r)
	var msgs []imapclient.Message
	var buf bytes.Buffer
	var uid uint32
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "From ") && buf.Len() > 0 {
			raw := append([]byte(nil), buf.Bytes()...)
			uid++
			msgs = append(msgs, imapclient.Message{UID: uid, Folder: "INBOX", Raw: raw, Hash: hash(raw), Subject: parseSubject(raw)})
			buf.Reset()
			continue
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
	}
	if buf.Len() > 0 {
		raw := append([]byte(nil), buf.Bytes()...)
		uid++
		msgs = append(msgs, imapclient.Message{UID: uid, Folder: "INBOX", Raw: raw, Hash: hash(raw), Subject: parseSubject(raw)})
	}
	return msgs, sc.Err()
}

// ImportPST reads a minimal PST subset (headers smoke); bulk PST via external tool.
func ImportPST(path string) ([]imapclient.Message, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 4 || string(b[:4]) != "!BDN" {
		return nil, fmt.Errorf("pst: not a PST header (use external export for bulk)")
	}
	// Smoke: scan for embedded RFC822 blocks
	var msgs []imapclient.Message
	parts := bytes.Split(b, []byte("From:"))
	for i, p := range parts {
		if i == 0 || len(p) < 10 {
			continue
		}
		raw := append([]byte("From:"), p...)
		if idx := bytes.Index(raw, []byte("\r\n\r\n")); idx < 0 {
			continue
		}
		msgs = append(msgs, imapclient.Message{UID: uint32(i), Folder: "INBOX", Raw: raw, Hash: hash(raw)})
		if len(msgs) >= 100 {
			break
		}
	}
	return msgs, nil
}

func hash(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func parseSubject(raw []byte) string {
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.ToLower(line), "subject:") {
			return strings.TrimSpace(line[8:])
		}
	}
	return ""
}

func ImportSmoke(filename string) bool {
	name := strings.ToLower(filename)
	return strings.HasSuffix(name, ".pst") || strings.HasSuffix(name, ".mbox")
}
