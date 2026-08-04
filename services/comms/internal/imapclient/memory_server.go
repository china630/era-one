package imapclient

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

// SeedMessage — test fixture with optional \Seen.
type SeedMessage struct {
	Raw  []byte
	Seen bool
}

// StartTestServer starts in-memory IMAP server for tests (single INBOX).
func StartTestServer(messages map[string][]byte) (addr string, stop func(), err error) {
	folders := map[string][]SeedMessage{"INBOX": {}}
	for _, raw := range messages {
		folders["INBOX"] = append(folders["INBOX"], SeedMessage{Raw: raw})
	}
	return startMemoryServer("127.0.0.1:0", folders, nil)
}

// StartTestServerFolders starts server with per-folder messages and optional LIST attrs.
func StartTestServerFolders(folders map[string][]SeedMessage, listAttrs map[string][]string) (addr string, stop func(), err error) {
	if folders == nil {
		folders = map[string][]SeedMessage{}
	}
	return startMemoryServer("127.0.0.1:0", folders, listAttrs)
}

// StartLabServer listens on addr (e.g. ":143") for compose lab IMAP (L-1 dovecot-lab).
func StartLabServer(addr string, folders map[string][]SeedMessage, listAttrs map[string][]string) (bound string, stop func(), err error) {
	if folders == nil {
		folders = map[string][]SeedMessage{"INBOX": {}}
	}
	if addr == "" {
		addr = ":143"
	}
	return startMemoryServer(addr, folders, listAttrs)
}

func startMemoryServer(listenAddr string, folders map[string][]SeedMessage, listAttrs map[string][]string) (string, func(), error) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return "", nil, err
	}
	srv := &memServer{
		folders:   map[string][]message{},
		listAttrs: listAttrs,
	}
	for name, seeds := range folders {
		if _, ok := srv.folders[name]; !ok {
			srv.folders[name] = nil
		}
		for i, seed := range seeds {
			srv.folders[name] = append(srv.folders[name], message{
				uid:  uint32(i + 1),
				raw:  seed.Raw,
				seen: seed.Seen,
			})
		}
	}
	if len(srv.folders) == 0 {
		srv.folders["INBOX"] = nil
	}
	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-stopCh:
					return
				default:
					continue
				}
			}
			go srv.handle(conn)
		}
	}()
	stop := func() {
		close(stopCh)
		_ = ln.Close()
		wg.Wait()
	}
	return ln.Addr().String(), stop, nil
}

type message struct {
	uid  uint32
	raw  []byte
	seen bool
}

type memServer struct {
	mu        sync.Mutex
	folders   map[string][]message
	selected  string
	listAttrs map[string][]string
}

func (s *memServer) handle(conn net.Conn) {
	defer conn.Close()
	_, _ = fmt.Fprintf(conn, "* OK era test IMAP\r\n")
	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		tag, cmdLine := parts[0], parts[1]
		upper := strings.ToUpper(cmdLine)
		switch {
		case strings.HasPrefix(upper, "LOGIN"):
			_, _ = fmt.Fprintf(conn, "%s OK LOGIN completed\r\n", tag)
		case strings.HasPrefix(upper, "LIST"):
			s.mu.Lock()
			names := make([]string, 0, len(s.folders))
			for name := range s.folders {
				names = append(names, name)
			}
			attrsCopy := s.listAttrs
			s.mu.Unlock()
			for _, name := range names {
				attrs := attrsCopy[name]
				_, _ = fmt.Fprintf(conn, "* LIST (%s) \"/\" \"%s\"\r\n", strings.Join(attrs, " "), name)
			}
			_, _ = fmt.Fprintf(conn, "%s OK LIST completed\r\n", tag)
		case strings.HasPrefix(upper, "SELECT"):
			folder := parseQuotedMailbox(cmdLine)
			if folder == "" {
				folder = "INBOX"
			}
			s.mu.Lock()
			s.selected = folder
			n := len(s.folders[folder])
			s.mu.Unlock()
			_, _ = fmt.Fprintf(conn, "* %d EXISTS\r\n", n)
			_, _ = fmt.Fprintf(conn, "%s OK SELECT completed\r\n", tag)
		case strings.HasPrefix(upper, "CREATE"):
			folder := parseQuotedMailbox(cmdLine)
			s.mu.Lock()
			if _, ok := s.folders[folder]; !ok {
				s.folders[folder] = nil
			}
			s.mu.Unlock()
			_, _ = fmt.Fprintf(conn, "%s OK CREATE completed\r\n", tag)
		case strings.HasPrefix(upper, "FETCH"):
			s.mu.Lock()
			msgs := append([]message(nil), s.folders[s.selected]...)
			s.mu.Unlock()
			for i, m := range msgs {
				flags := ""
				if m.seen {
					flags = "FLAGS (\\Seen) "
				}
				_, _ = fmt.Fprintf(conn, "* %d FETCH (%sUID %d BODY[] {%d}\r\n", i+1, flags, m.uid, len(m.raw))
				_, _ = conn.Write(m.raw)
				_, _ = fmt.Fprintf(conn, "\r\n)\r\n")
			}
			_, _ = fmt.Fprintf(conn, "%s OK FETCH completed\r\n", tag)
		case strings.HasPrefix(upper, "APPEND"):
			folder := parseAppendMailbox(cmdLine)
			seen := strings.Contains(strings.ToUpper(cmdLine), `\SEEN`)
			size := 0
			if i := strings.Index(cmdLine, "{"); i >= 0 {
				if j := strings.Index(cmdLine[i:], "}"); j >= 0 {
					size, _ = strconv.Atoi(strings.TrimSpace(cmdLine[i+1 : i+j]))
				}
			}
			_, _ = fmt.Fprintf(conn, "+ Ready for literal data\r\n")
			raw := make([]byte, size)
			if size > 0 {
				_, _ = ioReadFull(r, raw)
				_, _ = r.ReadString('\n')
			}
			s.mu.Lock()
			if folder == "" {
				folder = s.selected
			}
			if folder == "" {
				folder = "INBOX"
			}
			nextUID := uint32(len(s.folders[folder]) + 1)
			s.folders[folder] = append(s.folders[folder], message{uid: nextUID, raw: raw, seen: seen})
			s.mu.Unlock()
			_, _ = fmt.Fprintf(conn, "%s OK APPEND completed\r\n", tag)
		case strings.HasPrefix(upper, "LOGOUT"):
			_, _ = fmt.Fprintf(conn, "%s OK LOGOUT completed\r\n", tag)
			return
		default:
			_, _ = fmt.Fprintf(conn, "%s OK\r\n", tag)
		}
	}
}

func parseQuotedMailbox(cmdLine string) string {
	parts := strings.Split(cmdLine, `"`)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func parseAppendMailbox(cmdLine string) string {
	upper := strings.ToUpper(cmdLine)
	idx := strings.Index(upper, "APPEND")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(cmdLine[idx+6:])
	if strings.HasPrefix(rest, `"`) {
		if end := strings.Index(rest[1:], `"`); end >= 0 {
			return rest[1 : end+1]
		}
	}
	return ""
}

func ioReadFull(r *bufio.Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := r.Read(buf[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
