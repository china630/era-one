package repo

// Backend is the persistence interface used by mail-api and adapters.
type Backend interface {
	CreateMailbox(tenantID, email, password string, quotaBytes int64) (*Mailbox, error)
	UpdateMailbox(email string, patch MailboxPatch) (*Mailbox, error)
	ListMailboxes(tenantID string) ([]*Mailbox, error)
	GetMailboxByEmail(email string) (*Mailbox, error)
	VerifyMailboxPassword(email, password string) bool
	DeliverRaw(email string, raw []byte, fromAddr string) (*Message, error)
	ListMessages(email string) ([]*Message, error)
	AddEWSMessage(email, subject, body string) (*Message, error)
	GetMessageByID(email, msgID string) (*Message, bool)
	PutCalendarEvent(owner, uid, body string) (*CalendarEvent, error)
	GetCalendarEvent(owner, uid string) (*CalendarEvent, bool)
	ListCalendarEvents(owner string) ([]*CalendarEvent, error)
	FindCalendarEventByUID(uid string) (*CalendarEvent, bool)
	PutContact(owner, uid, vcard string) (*Contact, error)
	ListContacts(owner string) ([]*Contact, error)
	GetPolicy(tenantID string) (InlinePolicy, bool)
	PutPolicy(tenantID string, p InlinePolicy)
	GetEASSyncKey(deviceID, mailboxID, folderID string) (string, bool)
	PutEASSyncKey(deviceID, mailboxID, folderID, syncKey string) error
	MessageCount() int64
	Close() error
}

// Repository wraps Backend for dependency injection.
type Repository struct {
	Backend
}

// New returns a Repository from env.
func New() (*Repository, error) {
	b, err := OpenFromEnv()
	if err != nil {
		return nil, err
	}
	return &Repository{Backend: b}, nil
}

// Pingable allows readiness probes.
type Pingable interface {
	Ping() error
}

func (r *Repository) Ping() error {
	if p, ok := r.Backend.(Pingable); ok {
		return p.Ping()
	}
	return nil
}

func (r *Repository) PingBlob() error {
	if p, ok := r.Backend.(interface{ PingBlob() error }); ok {
		return p.PingBlob()
	}
	return nil
}
