// Package calendaraudit — calendar audit bridge to ClickHouse (F-C13b).
package calendaraudit

import (
	"context"

	"era/services/comms/mail/internal/audit"
	erav1 "era/contracts/gen/era/v1"
)

// Recorder implements caldav.Auditor.
type Recorder struct {
	Writer *audit.Writer
}

// RecordCalendar writes calendar create/update audit events.
func (r *Recorder) RecordCalendar(ctx context.Context, create bool, owner, uid string) error {
	if r == nil || r.Writer == nil {
		return nil
	}
	action := erav1.MailAuditAction_MAIL_AUDIT_CALENDAR_UPDATE
	if create {
		action = erav1.MailAuditAction_MAIL_AUDIT_CALENDAR_CREATE
	}
	return r.Writer.Insert(ctx, &erav1.MailAuditEvent{
		TenantId:  "t-demo",
		Mailbox:   owner,
		Action:    action,
		MessageId: uid,
		Metadata:  map[string]string{"calendar_uid": uid},
	})
}
