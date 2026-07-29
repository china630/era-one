package imapclient

import "strings"

// Mailbox — IMAP LIST entry with RFC 6154 special-use attributes when present.
type Mailbox struct {
	Name       string
	Attributes []string
}

// HasAttribute reports LIST flag (e.g. \Sent, \Noselect).
func (m Mailbox) HasAttribute(flag string) bool {
	want := strings.ToUpper(strings.TrimSpace(flag))
	for _, a := range m.Attributes {
		if strings.ToUpper(a) == want {
			return true
		}
	}
	return false
}

// Selectable returns false for \Noselect mailboxes.
func (m Mailbox) Selectable() bool {
	return !m.HasAttribute(`\Noselect`)
}
