package sync

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"era/services/comms/internal/imapclient"
)

// ResolvePassword — vault/env lookup for password_ref (air-gap: no plaintext in API).
// Supports: env:NAME, vault://NAME → ERA_CONNECT_SECRET_NAME, or raw ERA_CONNECT_SECRET_<NAME>.
func ResolvePassword(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("empty password_ref")
	}
	if strings.HasPrefix(ref, "env:") {
		v := os.Getenv(ref[4:])
		if v == "" {
			return "", fmt.Errorf("env %s empty", ref[4:])
		}
		return v, nil
	}
	name := ref
	name = strings.TrimPrefix(name, "vault://")
	name = strings.TrimPrefix(name, "secret://")
	key := "ERA_CONNECT_SECRET_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	if v := os.Getenv(key); v != "" {
		return v, nil
	}
	if v := os.Getenv("ERA_CONNECT_SECRET_" + name); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("secret not found for ref %q (set %s)", ref, key)
}

// Dialer — injectable for tests.
type Dialer func(cfg imapclient.Config) (*imapclient.Client, error)

func defaultDial(cfg imapclient.Config) (*imapclient.Client, error) {
	return imapclient.Dial(cfg)
}

// StartSync performs real IMAP FETCH when mailbox.Address is set and password resolves;
// otherwise returns honest mode=stub with ItemsOK=0 (no silent inflate).
func (s *Store) StartSync(tenantID, mailbox string) Job {
	return s.StartSyncWith(tenantID, mailbox, defaultDial)
}

func (s *Store) StartSyncWith(tenantID, mailbox string, dial Dialer) Job {
	s.mu.Lock()
	mb, ok := s.mailbox[key(tenantID, mailbox)]
	s.sequence++
	id := "sync-" + time.Now().UTC().Format("20060102150405") + "-" + strconv.Itoa(s.sequence)
	s.mu.Unlock()

	j := Job{
		ID:        id,
		Mailbox:   mailbox,
		TenantID:  tenantID,
		Status:    "running",
		CreatedAt: time.Now().UTC(),
	}

	if !ok || strings.TrimSpace(mb.Address) == "" {
		// Honest stub: no silent ItemsOK=12 inflate (G0-7 / AC-C6)
		j.Status = "done"
		j.Mode = "stub"
		j.ItemsTotal = 0
		j.ItemsOK = 0
		j.Error = "no mailbox address; mode=stub"
		s.mu.Lock()
		s.jobs[j.ID] = j
		s.mu.Unlock()
		return j
	}

	pass, err := ResolvePassword(mb.PasswordRef)
	if err != nil {
		j.Status = "failed"
		j.ItemsFail = 1
		j.Error = err.Error()
		s.mu.Lock()
		s.jobs[j.ID] = j
		s.mu.Unlock()
		return j
	}

	host, port, tlsOn := parseAddress(mb.Address)
	user := mb.Username
	if user == "" {
		user = mb.Email
	}
	cl, err := dial(imapclient.Config{
		Host: host, Port: port, Username: user, Password: pass, TLS: tlsOn,
	})
	if err != nil {
		j.Status = "failed"
		j.ItemsFail = 1
		j.Error = err.Error()
		s.mu.Lock()
		s.jobs[j.ID] = j
		s.mu.Unlock()
		return j
	}
	defer cl.Close()

	j.ItemsTotal = 0
	j.ItemsOK = 0
	j.Mode = "live"
	folders := []string{"INBOX", "Sent", "Sent Items"}
	fetched := 0
	var folderErrs []string
	for _, folder := range folders {
		msgs, err := cl.FetchFolder(folder)
		if err != nil {
			folderErrs = append(folderErrs, folder+": "+err.Error())
			continue
		}
		fetched++
		j.ItemsTotal += len(msgs)
		j.ItemsOK += len(msgs)
		var maxUID uint32
		for _, m := range msgs {
			if m.UID > maxUID {
				maxUID = m.UID
			}
		}
		s.mu.Lock()
		if s.cursors == nil {
			s.cursors = map[string]uint32{}
		}
		s.cursors[key(tenantID, mailbox)+":"+folder] = maxUID
		s.mu.Unlock()
	}
	if fetched == 0 {
		j.Status = "failed"
		j.ItemsFail = 1
		j.Error = "no folders fetched"
		if len(folderErrs) > 0 {
			j.Error += "; " + strings.Join(folderErrs, "; ")
		}
		s.mu.Lock()
		s.jobs[j.ID] = j
		s.mu.Unlock()
		return j
	}
	j.Status = "done"
	s.mu.Lock()
	s.jobs[j.ID] = j
	s.mu.Unlock()
	return j
}

func parseAddress(addr string) (host string, port int, tls bool) {
	host = addr
	port = 143
	tls = false
	if strings.HasPrefix(host, "imaps://") {
		tls = true
		port = 993
		host = strings.TrimPrefix(host, "imaps://")
	} else if strings.HasPrefix(host, "imap://") {
		host = strings.TrimPrefix(host, "imap://")
	}
	if h, p, err := net.SplitHostPort(host); err == nil {
		host = h
		if n, e := strconv.Atoi(p); e == nil {
			port = n
			if port == 993 {
				tls = true
			}
		}
	} else if strings.Contains(host, ":") {
		// bare host:port without brackets
		parts := strings.Split(host, ":")
		if len(parts) == 2 {
			host = parts[0]
			if n, e := strconv.Atoi(parts[1]); e == nil {
				port = n
				if port == 993 {
					tls = true
				}
			}
		}
	}
	return host, port, tls
}
