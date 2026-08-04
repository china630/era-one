package repo

// MailboxPatch updates mailbox fields.
type MailboxPatch struct {
	Password   *string `json:"password,omitempty"`
	QuotaBytes *int64  `json:"quota_bytes,omitempty"`
	Active     *bool   `json:"active,omitempty"`
}
