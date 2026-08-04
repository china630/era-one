package repo

import "time"

// Mailbox is a mail account.
type Mailbox struct {
	ID         string
	TenantID   string
	Email      string
	QuotaBytes int64
	UsedBytes  int64
	Active     bool
}

// Message is a stored mail item.
type Message struct {
	ID        int64
	MailboxID string
	Email     string
	UID       int64
	Subject   string
	FromAddr  string
	Body      string
	Raw       []byte
	MinioKey  string
	Flags     string
	CreatedAt time.Time
}

// CalendarEvent is a CalDAV/iCal object.
type CalendarEvent struct {
	ID        string
	Owner     string
	UID       string
	Body      string
	ETag      string
	UpdatedAt time.Time
}

// Contact is a CardDAV vCard.
type Contact struct {
	ID        string
	Owner     string
	UID       string
	VCard     string
	ETag      string
	UpdatedAt time.Time
}

// InlinePolicy mirrors policy.InlinePolicy for persistence.
type InlinePolicy struct {
	MaxAttachmentSizeMB      uint32
	QuotaMBPerUser           uint32
	RetentionDays            uint32
	MaxAttachmentsPerMessage uint32
	MaxRecipients            int
	AttachmentExtDeny        []string
}
