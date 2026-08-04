// Package policy — tenant policy для inline attachments (PRD §3, AC-C4).
package policy

import "sync"

// InlinePolicy — квоты вложений без ERA Drive.
type InlinePolicy struct {
	MaxAttachmentSizeMB      uint32 `json:"max_attachment_size_mb"`
	QuotaMBPerUser           uint32 `json:"quota_mb_per_user"`
	RetentionDays            uint32 `json:"retention_days"`
	MaxAttachmentsPerMessage uint32 `json:"max_attachments_per_message"`
}

// DefaultPolicy возвращает рекомендованные defaults из PRD §3.
func DefaultPolicy() InlinePolicy {
	return InlinePolicy{
		MaxAttachmentSizeMB:      25,
		QuotaMBPerUser:           512,
		RetentionDays:            365,
		MaxAttachmentsPerMessage: 50,
	}
}

// Store — in-memory tenant policies (MVP; позже — control-plane sync).
type Store struct {
	mu   sync.RWMutex
	data map[string]InlinePolicy
}

// NewStore создаёт пустой policy store.
func NewStore() *Store {
	return &Store{data: make(map[string]InlinePolicy)}
}

// Put сохраняет policy tenant.
func (s *Store) Put(tenantID string, p InlinePolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[tenantID] = p
}

// Get возвращает policy tenant; ok=false если не задана.
func (s *Store) Get(tenantID string) (InlinePolicy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[tenantID]
	return p, ok
}
