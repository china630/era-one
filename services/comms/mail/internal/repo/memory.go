package repo

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Memory is an in-memory Repository (unit tests, ERA_COMMS_DATABASE_URL empty).
type Memory struct {
	mu         sync.RWMutex
	mailboxes  map[string]*mailboxRec
	byEmail    map[string]string
	messages   map[string][]*Message
	calendars  map[string]map[string]*CalendarEvent
	contacts   map[string]map[string]*Contact
	policies   map[string]InlinePolicy
	easKeys    map[string]string
	msgSeq     int64
	uidCounter map[string]int64
}

type mailboxRec struct {
	Mailbox
	PasswordHash string
}

// NewMemory returns empty in-memory repo.
func NewMemory() *Memory {
	return &Memory{
		mailboxes:  make(map[string]*mailboxRec),
		byEmail:    make(map[string]string),
		messages:   make(map[string][]*Message),
		calendars:  make(map[string]map[string]*CalendarEvent),
		contacts:   make(map[string]map[string]*Contact),
		policies:   make(map[string]InlinePolicy),
		easKeys:    make(map[string]string),
		uidCounter: make(map[string]int64),
	}
}

func (m *Memory) CreateMailbox(tenantID, email, password string, quotaBytes int64) (*Mailbox, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("invalid email")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byEmail[email]; ok {
		return nil, fmt.Errorf("mailbox exists")
	}
	id := uuid.NewString()
	mb := &mailboxRec{
		Mailbox: Mailbox{
			ID:         id,
			TenantID:   tenantID,
			Email:      email,
			QuotaBytes: quotaBytes,
			Active:     true,
		},
		PasswordHash: hash,
	}
	m.mailboxes[id] = mb
	m.byEmail[email] = id
	return &mb.Mailbox, nil
}

func (m *Memory) UpdateMailbox(email string, patch MailboxPatch) (*Mailbox, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	mb := m.mailboxes[id]
	if patch.Password != nil {
		hash, err := HashPassword(*patch.Password)
		if err != nil {
			return nil, err
		}
		mb.PasswordHash = hash
	}
	if patch.QuotaBytes != nil {
		mb.QuotaBytes = *patch.QuotaBytes
	}
	if patch.Active != nil {
		mb.Active = *patch.Active
	}
	out := mb.Mailbox
	return &out, nil
}

func (m *Memory) ListMailboxes(tenantID string) ([]*Mailbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Mailbox
	for _, mb := range m.mailboxes {
		if mb.TenantID == tenantID {
			copy := mb.Mailbox
			out = append(out, &copy)
		}
	}
	return out, nil
}

func (m *Memory) GetMailboxByEmail(email string) (*Mailbox, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return &m.mailboxes[id].Mailbox, nil
}

func (m *Memory) VerifyMailboxPassword(email, password string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byEmail[email]
	if !ok {
		return false
	}
	return VerifyPassword(m.mailboxes[id].PasswordHash, password)
}

func (m *Memory) DeliverRaw(email string, raw []byte, fromAddr string) (*Message, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("mailbox not found")
	}
	mb := m.mailboxes[id]
	if pol, ok := m.policies[mb.TenantID]; ok && pol.MaxAttachmentSizeMB > 0 {
		max := int64(pol.MaxAttachmentSizeMB) * 1024 * 1024
		if int64(len(raw)) > max {
			return nil, fmt.Errorf("message too large")
		}
	}
	if int64(len(raw)) > mb.QuotaBytes-mb.UsedBytes {
		return nil, fmt.Errorf("quota exceeded")
	}
	m.msgSeq++
	m.uidCounter[id]++
	msg := &Message{
		ID:        m.msgSeq,
		MailboxID: id,
		Email:     email,
		UID:       m.uidCounter[id],
		Raw:       append([]byte(nil), raw...),
		FromAddr:  fromAddr,
		Flags:     "",
		CreatedAt: time.Now().UTC(),
	}
	m.messages[id] = append(m.messages[id], msg)
	mb.UsedBytes += int64(len(raw))
	return msg, nil
}

func (m *Memory) ListMessages(email string) ([]*Message, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	src := m.messages[id]
	out := make([]*Message, len(src))
	copy(out, src)
	return out, nil
}

func (m *Memory) AddEWSMessage(email, subject, body string) (*Message, error) {
	raw := []byte("Subject: " + subject + "\r\n\r\n" + body)
	msg, err := m.DeliverRaw(email, raw, email)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	msg.Subject = subject
	msg.Body = body
	m.mu.Unlock()
	return msg, nil
}

func (m *Memory) GetMessageByID(email, msgID string) (*Message, bool) {
	msgs, err := m.ListMessages(email)
	if err != nil {
		return nil, false
	}
	for _, msg := range msgs {
		if formatMsgID(msg.ID) == msgID || fmt.Sprintf("%d", msg.ID) == msgID {
			return msg, true
		}
	}
	return nil, false
}

func formatMsgID(id int64) string {
	return fmt.Sprintf("msg-%d", id)
}

func (m *Memory) PutCalendarEvent(owner, uid, body string) (*CalendarEvent, error) {
	owner = strings.ToLower(strings.TrimSpace(owner))
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.calendars[owner] == nil {
		m.calendars[owner] = make(map[string]*CalendarEvent)
	}
	ev := &CalendarEvent{
		ID:        uuid.NewString(),
		Owner:     owner,
		UID:       uid,
		Body:      body,
		ETag:      fmt.Sprintf("\"%d\"", time.Now().UnixNano()),
		UpdatedAt: time.Now().UTC(),
	}
	m.calendars[owner][uid] = ev
	return ev, nil
}

func (m *Memory) GetCalendarEvent(owner, uid string) (*CalendarEvent, bool) {
	owner = strings.ToLower(strings.TrimSpace(owner))
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.calendars[owner] == nil {
		return nil, false
	}
	ev, ok := m.calendars[owner][uid]
	return ev, ok
}

func (m *Memory) ListCalendarEvents(owner string) ([]*CalendarEvent, error) {
	owner = strings.ToLower(strings.TrimSpace(owner))
	m.mu.RLock()
	defer m.mu.RUnlock()
	events := m.calendars[owner]
	out := make([]*CalendarEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, ev)
	}
	return out, nil
}

func (m *Memory) FindCalendarEventByUID(uid string) (*CalendarEvent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for owner, events := range m.calendars {
		if ev, ok := events[uid]; ok {
			copy := *ev
			copy.Owner = owner
			return &copy, true
		}
	}
	return nil, false
}

func (m *Memory) PutContact(owner, uid, vcard string) (*Contact, error) {
	owner = strings.ToLower(strings.TrimSpace(owner))
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.contacts[owner] == nil {
		m.contacts[owner] = make(map[string]*Contact)
	}
	c := &Contact{
		ID:        uuid.NewString(),
		Owner:     owner,
		UID:       uid,
		VCard:     vcard,
		ETag:      fmt.Sprintf("\"%d\"", time.Now().UnixNano()),
		UpdatedAt: time.Now().UTC(),
	}
	m.contacts[owner][uid] = c
	return c, nil
}

func (m *Memory) ListContacts(owner string) ([]*Contact, error) {
	owner = strings.ToLower(strings.TrimSpace(owner))
	m.mu.RLock()
	defer m.mu.RUnlock()
	contacts := m.contacts[owner]
	out := make([]*Contact, 0, len(contacts))
	for _, c := range contacts {
		out = append(out, c)
	}
	return out, nil
}

func (m *Memory) GetPolicy(tenantID string) (InlinePolicy, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.policies[tenantID]
	return p, ok
}

func (m *Memory) PutPolicy(tenantID string, p InlinePolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[tenantID] = p
}

func (m *Memory) GetEASSyncKey(deviceID, mailboxID, folderID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key, ok := m.easKeys[deviceID+"|"+mailboxID+"|"+folderID]
	return key, ok
}

func (m *Memory) PutEASSyncKey(deviceID, mailboxID, folderID, syncKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.easKeys[deviceID+"|"+mailboxID+"|"+folderID] = syncKey
	return nil
}

func (m *Memory) MessageCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.msgSeq
}

func (m *Memory) Close() error { return nil }
