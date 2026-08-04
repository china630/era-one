package policy

import "testing"

func TestDefaultPolicyMatchesPRD(t *testing.T) {
	p := DefaultPolicy()
	if p.MaxAttachmentSizeMB != 25 {
		t.Fatalf("max_attachment_size_mb: %d", p.MaxAttachmentSizeMB)
	}
	if p.QuotaMBPerUser != 512 {
		t.Fatalf("quota_mb_per_user: %d", p.QuotaMBPerUser)
	}
	if p.RetentionDays != 365 {
		t.Fatalf("retention_days: %d", p.RetentionDays)
	}
	if p.MaxAttachmentsPerMessage != 50 {
		t.Fatalf("max_attachments_per_message: %d", p.MaxAttachmentsPerMessage)
	}
}

func TestStorePutGet(t *testing.T) {
	s := NewStore()
	custom := DefaultPolicy()
	custom.MaxAttachmentSizeMB = 10
	s.Put("t1", custom)
	got, ok := s.Get("t1")
	if !ok || got.MaxAttachmentSizeMB != 10 {
		t.Fatalf("unexpected: %+v ok=%v", got, ok)
	}
}
