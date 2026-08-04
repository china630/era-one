package imapclient

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// FetchFolder returns messages in folder.
func (cl *Client) FetchFolder(folder string) ([]Message, error) {
	if _, err := cl.cmd(`SELECT "%s"`, escape(folder)); err != nil {
		return nil, err
	}
	tag := cl.nextTag()
	if _, err := fmt.Fprintf(cl.conn, "%s FETCH 1:* (UID FLAGS BODY[])\r\n", tag); err != nil {
		return nil, err
	}
	var out []Message
	for {
		line, err := cl.readLine()
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(line, tag+" OK") {
			break
		}
		if !strings.Contains(line, "FETCH") || !strings.Contains(line, "BODY[]") {
			continue
		}
		uid := parseUID(line)
		size := parseLiteralSize(line)
		if size <= 0 {
			continue
		}
		raw := make([]byte, size)
		if _, err := io.ReadFull(cl.reader, raw); err != nil {
			return nil, err
		}
		if _, err := cl.readLine(); err != nil { // trailing )
			return nil, err
		}
		out = append(out, Message{
			UID:     uid,
			Folder:  folder,
			Raw:     raw,
			Hash:    HashRaw(raw),
			Subject: parseSubject(raw),
			Seen:    parseSeen(line),
		})
	}
	return out, nil
}

func parseUID(line string) uint32 {
	idx := strings.Index(line, "UID ")
	if idx < 0 {
		return 0
	}
	rest := strings.TrimSpace(line[idx+4:])
	end := strings.IndexAny(rest, " )")
	if end <= 0 {
		return 0
	}
	n, _ := strconv.ParseUint(rest[:end], 10, 32)
	return uint32(n)
}

func parseLiteralSize(line string) int {
	i := strings.LastIndex(line, "{")
	if i < 0 {
		return 0
	}
	j := strings.Index(line[i:], "}")
	if j < 0 {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(line[i+1 : i+j]))
	return n
}

func parseSeen(line string) bool {
	return strings.Contains(strings.ToUpper(line), `\SEEN`)
}
