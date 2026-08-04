package jobs

// Repository abstracts job storage (memory or Postgres).
type Repository interface {
	CreateQueued(source, mailbox string) Job
	CreateDone(source, mailbox string, total int) Job
	Get(id string) (Job, bool)
	SetStatus(id, status string)
	Complete(id string, total, ok, fail int)
	Fail(id, errMsg string)
	MarkSeen(uid string)
	Seen(uid string) bool
	Rerun(sourceUIDs []string) int
	DequeueQueued() (Job, bool)
}

// Ensure Store implements Repository.
var _ Repository = (*Store)(nil)
