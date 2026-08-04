package orchestrator

import (
	"era/services/comms/migration/internal/jobs"
)

// Queue exposes dequeue for worker pool (PG-backed).
type Queue struct {
	repo jobs.Repository
}

func NewQueue(repo jobs.Repository) *Queue { return &Queue{repo: repo} }

func (q *Queue) Next() (jobs.Job, bool) { return q.repo.DequeueQueued() }
