package repo

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"era/services/comms/mail/internal/blobstore"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed schema.sql
var schemaSQL string

// Postgres persists data in era_comms schema.
type Postgres struct {
	db    *sql.DB
	blobs blobstore.Store
}

// OpenPostgres connects and applies schema if needed.
func OpenPostgres(dsn string) (*Postgres, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	blobs, _ := blobstore.OpenFromEnv()
	return &Postgres{db: db, blobs: blobs}, nil
}

// OpenFromEnv opens Postgres when ERA_COMMS_DATABASE_URL is set.
// Memory is only allowed when ERA_MAIL_STORE=memory (G1-1 honesty).
func OpenFromEnv() (Backend, error) {
	storeMode := strings.ToLower(strings.TrimSpace(os.Getenv("ERA_MAIL_STORE")))
	dsn := strings.TrimSpace(os.Getenv("ERA_COMMS_DATABASE_URL"))
	if storeMode == "memory" {
		return NewMemory(), nil
	}
	if dsn == "" {
		return nil, fmt.Errorf("ERA_COMMS_DATABASE_URL required (set ERA_MAIL_STORE=memory for in-memory lab)")
	}
	return OpenPostgres(dsn)
}

func (p *Postgres) Close() error { return p.db.Close() }

func (p *Postgres) ensureTenant(tenantID string) error {
	if tenantID == "" {
		tenantID = "t-demo"
	}
	_, err := p.db.Exec(
		`INSERT INTO era_comms.tenants (id, name, slug) VALUES ($1,$2,$3) ON CONFLICT (id) DO NOTHING`,
		tenantID, tenantID, tenantID,
	)
	return err
}

func (p *Postgres) CreateMailbox(tenantID, email, password string, quotaBytes int64) (*Mailbox, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	if tenantID == "" {
		tenantID = "t-demo"
	}
	if err := p.ensureTenant(tenantID); err != nil {
		return nil, err
	}
	id := uuid.NewString()
	_, err = p.db.Exec(
		`INSERT INTO era_comms.mailboxes (id, tenant_id, email, password_hash, quota_bytes) VALUES ($1,$2,$3,$4,$5)`,
		id, tenantID, email, hash, quotaBytes,
	)
	if err != nil {
		return nil, err
	}
	return &Mailbox{ID: id, TenantID: tenantID, Email: email, QuotaBytes: quotaBytes, Active: true}, nil
}

func (p *Postgres) UpdateMailbox(email string, patch MailboxPatch) (*Mailbox, error) {
	mb, err := p.GetMailboxByEmail(email)
	if err != nil {
		return nil, err
	}
	if patch.Password != nil {
		hash, err := HashPassword(*patch.Password)
		if err != nil {
			return nil, err
		}
		_, _ = p.db.Exec(`UPDATE era_comms.mailboxes SET password_hash=$1 WHERE id=$2`, hash, mb.ID)
	}
	if patch.QuotaBytes != nil {
		_, _ = p.db.Exec(`UPDATE era_comms.mailboxes SET quota_bytes=$1 WHERE id=$2`, *patch.QuotaBytes, mb.ID)
		mb.QuotaBytes = *patch.QuotaBytes
	}
	if patch.Active != nil {
		_, _ = p.db.Exec(`UPDATE era_comms.mailboxes SET active=$1 WHERE id=$2`, *patch.Active, mb.ID)
		mb.Active = *patch.Active
	}
	return p.GetMailboxByEmail(email)
}

func (p *Postgres) ListMailboxes(tenantID string) ([]*Mailbox, error) {
	rows, err := p.db.Query(
		`SELECT id, tenant_id, email, quota_bytes, used_bytes, active FROM era_comms.mailboxes WHERE tenant_id=$1 ORDER BY email`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Mailbox
	for rows.Next() {
		var mb Mailbox
		if err := rows.Scan(&mb.ID, &mb.TenantID, &mb.Email, &mb.QuotaBytes, &mb.UsedBytes, &mb.Active); err != nil {
			return nil, err
		}
		out = append(out, &mb)
	}
	return out, rows.Err()
}

func (p *Postgres) GetMailboxByEmail(email string) (*Mailbox, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var mb Mailbox
	err := p.db.QueryRow(
		`SELECT id, tenant_id, email, quota_bytes, used_bytes, active FROM era_comms.mailboxes WHERE email=$1`, email,
	).Scan(&mb.ID, &mb.TenantID, &mb.Email, &mb.QuotaBytes, &mb.UsedBytes, &mb.Active)
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	return &mb, nil
}

func (p *Postgres) VerifyMailboxPassword(email, password string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	var hash string
	err := p.db.QueryRow(`SELECT password_hash FROM era_comms.mailboxes WHERE email=$1 AND active`, email).Scan(&hash)
	if err != nil {
		return false
	}
	return VerifyPassword(hash, password)
}

func (p *Postgres) DeliverRaw(email string, raw []byte, fromAddr string) (*Message, error) {
	mb, err := p.GetMailboxByEmail(email)
	if err != nil {
		return nil, err
	}
	if pol, ok := p.GetPolicy(mb.TenantID); ok {
		max := int64(pol.MaxAttachmentSizeMB) * 1024 * 1024
		if int64(len(raw)) > max {
			return nil, fmt.Errorf("message too large")
		}
	}
	if int64(len(raw)) > mb.QuotaBytes-mb.UsedBytes {
		return nil, fmt.Errorf("quota exceeded")
	}
	var uid int64
	err = p.db.QueryRow(`SELECT COALESCE(MAX(uid),0)+1 FROM era_comms.messages WHERE mailbox_id=$1`, mb.ID).Scan(&uid)
	if err != nil {
		return nil, err
	}
	var minioKey string
	var inline []byte
	threshold := blobstore.ThresholdBytes()
	if p.blobs != nil && len(raw) > threshold {
		key, err := p.blobs.Put(raw)
		if err != nil {
			return nil, err
		}
		minioKey = key
	} else {
		inline = raw
	}
	var id int64
	err = p.db.QueryRow(
		`INSERT INTO era_comms.messages (mailbox_id, uid, from_addr, raw_inline, size_bytes, minio_key) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		mb.ID, uid, fromAddr, inline, len(raw), nullString(minioKey),
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	_, _ = p.db.Exec(`UPDATE era_comms.mailboxes SET used_bytes=used_bytes+$1 WHERE id=$2`, len(raw), mb.ID)
	outRaw := raw
	if minioKey != "" {
		outRaw = nil
	}
	return &Message{ID: id, MailboxID: mb.ID, Email: email, UID: uid, Raw: outRaw, MinioKey: minioKey, FromAddr: fromAddr, CreatedAt: time.Now().UTC()}, nil
}

func (p *Postgres) ListMessages(email string) ([]*Message, error) {
	mb, err := p.GetMailboxByEmail(email)
	if err != nil {
		return nil, err
	}
	rows, err := p.db.Query(
		`SELECT id, uid, COALESCE(subject,''), COALESCE(from_addr,''), raw_inline, COALESCE(flags,''), COALESCE(minio_key,''), created_at
		 FROM era_comms.messages WHERE mailbox_id=$1 ORDER BY uid`, mb.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Message
	for rows.Next() {
		var m Message
		var raw []byte
		var minioKey string
		m.MailboxID = mb.ID
		m.Email = email
		if err := rows.Scan(&m.ID, &m.UID, &m.Subject, &m.FromAddr, &raw, &m.Flags, &minioKey, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Raw = raw
		m.MinioKey = minioKey
		if m.MinioKey != "" && p.blobs != nil {
			if blob, err := p.blobs.Get(m.MinioKey); err == nil {
				m.Raw = blob
			}
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (p *Postgres) AddEWSMessage(email, subject, body string) (*Message, error) {
	raw := []byte("Subject: " + subject + "\r\n\r\n" + body)
	msg, err := p.DeliverRaw(email, raw, email)
	if err != nil {
		return nil, err
	}
	_, _ = p.db.Exec(`UPDATE era_comms.messages SET subject=$1 WHERE id=$2`, subject, msg.ID)
	msg.Subject = subject
	msg.Body = body
	return msg, nil
}

func (p *Postgres) GetMessageByID(email, msgID string) (*Message, bool) {
	msgs, err := p.ListMessages(email)
	if err != nil {
		return nil, false
	}
	for _, msg := range msgs {
		if formatMsgID(msg.ID) == msgID {
			return msg, true
		}
	}
	return nil, false
}

func (p *Postgres) PutCalendarEvent(owner, uid, body string) (*CalendarEvent, error) {
	mb, err := p.GetMailboxByEmail(owner)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	etag := fmt.Sprintf("\"%d\"", time.Now().UnixNano())
	_, err = p.db.Exec(
		`INSERT INTO era_comms.calendar_events (id, mailbox_id, uid, ical_data, etag)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (mailbox_id, uid) DO UPDATE SET ical_data=EXCLUDED.ical_data, etag=EXCLUDED.etag, updated_at=NOW()`,
		id, mb.ID, uid, body, etag,
	)
	if err != nil {
		return nil, err
	}
	return &CalendarEvent{ID: id, Owner: owner, UID: uid, Body: body, ETag: etag, UpdatedAt: time.Now().UTC()}, nil
}

func (p *Postgres) GetCalendarEvent(owner, uid string) (*CalendarEvent, bool) {
	mb, err := p.GetMailboxByEmail(owner)
	if err != nil {
		return nil, false
	}
	var ev CalendarEvent
	err = p.db.QueryRow(
		`SELECT id, uid, ical_data, etag, updated_at FROM era_comms.calendar_events WHERE mailbox_id=$1 AND uid=$2`,
		mb.ID, uid,
	).Scan(&ev.ID, &ev.UID, &ev.Body, &ev.ETag, &ev.UpdatedAt)
	if err != nil {
		return nil, false
	}
	ev.Owner = owner
	return &ev, true
}

func (p *Postgres) ListCalendarEvents(owner string) ([]*CalendarEvent, error) {
	mb, err := p.GetMailboxByEmail(owner)
	if err != nil {
		return nil, err
	}
	rows, err := p.db.Query(
		`SELECT id, uid, ical_data, etag, updated_at FROM era_comms.calendar_events WHERE mailbox_id=$1`, mb.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*CalendarEvent
	for rows.Next() {
		var ev CalendarEvent
		ev.Owner = owner
		if err := rows.Scan(&ev.ID, &ev.UID, &ev.Body, &ev.ETag, &ev.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &ev)
	}
	return out, rows.Err()
}

func (p *Postgres) FindCalendarEventByUID(uid string) (*CalendarEvent, bool) {
	var ev CalendarEvent
	var ownerEmail string
	err := p.db.QueryRow(
		`SELECT ce.id, ce.uid, ce.ical_data, ce.etag, ce.updated_at, m.email
		 FROM era_comms.calendar_events ce
		 JOIN era_comms.mailboxes m ON m.id = ce.mailbox_id
		 WHERE ce.uid=$1 LIMIT 1`, uid,
	).Scan(&ev.ID, &ev.UID, &ev.Body, &ev.ETag, &ev.UpdatedAt, &ownerEmail)
	if err != nil {
		return nil, false
	}
	ev.Owner = ownerEmail
	return &ev, true
}

func (p *Postgres) PutContact(owner, uid, vcard string) (*Contact, error) {
	mb, err := p.GetMailboxByEmail(owner)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	etag := fmt.Sprintf("\"%d\"", time.Now().UnixNano())
	_, err = p.db.Exec(
		`INSERT INTO era_comms.contacts (id, mailbox_id, uid, vcard_data, etag)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (mailbox_id, uid) DO UPDATE SET vcard_data=EXCLUDED.vcard_data, etag=EXCLUDED.etag, updated_at=NOW()`,
		id, mb.ID, uid, vcard, etag,
	)
	if err != nil {
		return nil, err
	}
	return &Contact{ID: id, Owner: owner, UID: uid, VCard: vcard, ETag: etag, UpdatedAt: time.Now().UTC()}, nil
}

func (p *Postgres) ListContacts(owner string) ([]*Contact, error) {
	mb, err := p.GetMailboxByEmail(owner)
	if err != nil {
		return nil, err
	}
	rows, err := p.db.Query(
		`SELECT id, uid, vcard_data, etag, updated_at FROM era_comms.contacts WHERE mailbox_id=$1`, mb.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Contact
	for rows.Next() {
		var c Contact
		c.Owner = owner
		if err := rows.Scan(&c.ID, &c.UID, &c.VCard, &c.ETag, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (pg *Postgres) GetPolicy(tenantID string) (InlinePolicy, bool) {
	var pol InlinePolicy
	err := pg.db.QueryRow(
		`SELECT inline_max_attachment_mb, inline_quota_mb_per_user, inline_retention_days, inline_max_attachments
		 FROM era_comms.tenant_policies WHERE tenant_id=$1`, tenantID,
	).Scan(&pol.MaxAttachmentSizeMB, &pol.QuotaMBPerUser, &pol.RetentionDays, &pol.MaxAttachmentsPerMessage)
	if err != nil {
		return InlinePolicy{}, false
	}
	return pol, true
}

func (pg *Postgres) PutPolicy(tenantID string, pol InlinePolicy) {
	_ = pg.ensureTenant(tenantID)
	_, _ = pg.db.Exec(
		`INSERT INTO era_comms.tenant_policies (tenant_id, inline_max_attachment_mb, inline_quota_mb_per_user, inline_retention_days, inline_max_attachments)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   inline_max_attachment_mb=EXCLUDED.inline_max_attachment_mb,
		   inline_quota_mb_per_user=EXCLUDED.inline_quota_mb_per_user,
		   inline_retention_days=EXCLUDED.inline_retention_days,
		   inline_max_attachments=EXCLUDED.inline_max_attachments`,
		tenantID, pol.MaxAttachmentSizeMB, pol.QuotaMBPerUser, pol.RetentionDays, pol.MaxAttachmentsPerMessage,
	)
}

func (pg *Postgres) MessageCount() int64 {
	var n int64
	_ = pg.db.QueryRow(`SELECT COUNT(*) FROM era_comms.messages`).Scan(&n)
	return n
}

func (pg *Postgres) Ping() error {
	return pg.db.Ping()
}

func (pg *Postgres) PingBlob() error {
	if pg.blobs == nil {
		return nil
	}
	return pg.blobs.Ping()
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (pg *Postgres) GetEASSyncKey(deviceID, mailboxID, folderID string) (string, bool) {
	var key string
	err := pg.db.QueryRow(
		`SELECT sync_key FROM era_comms.eas_device_sync_keys WHERE device_id=$1 AND mailbox_id=$2 AND folder_id=$3`,
		deviceID, mailboxID, folderID,
	).Scan(&key)
	if err != nil {
		return "", false
	}
	return key, true
}

func (pg *Postgres) PutEASSyncKey(deviceID, mailboxID, folderID, syncKey string) error {
	_, err := pg.db.Exec(
		`INSERT INTO era_comms.eas_device_sync_keys (device_id, mailbox_id, folder_id, sync_key)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (device_id, mailbox_id, folder_id) DO UPDATE SET sync_key=EXCLUDED.sync_key, updated_at=NOW()`,
		deviceID, mailboxID, folderID, syncKey,
	)
	return err
}
