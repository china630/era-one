package orchestrator

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"era/services/comms/migration/internal/jobs"
	"era/services/comms/migration/internal/worker"
)

// Pool runs N workers dequeuing queued jobs from the repository.
type Pool struct {
	Workers int
	Jobs    jobs.Repository
	Runner  *worker.Runner
	Throttle *HostThrottle
	wg      sync.WaitGroup
}

func NewPool(repo jobs.Repository, runner *worker.Runner) *Pool {
	n := 8
	if os.Getenv("ERA_MIG_SCALE") == "1" {
		n = 200
	} else if os.Getenv("ERA_MIG_SCALE") == "pilot1k" {
		n = 48
	}
	if v := os.Getenv("ERA_MIG_WORKERS"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			n = p
		}
	}
	limit := 10
	if n >= 32 {
		limit = 50
	}
	return &Pool{
		Workers:  n,
		Jobs:     repo,
		Runner:   runner,
		Throttle: NewHostThrottle(limit, time.Second),
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.Workers; i++ {
		p.wg.Add(1)
		go p.loop(ctx)
	}
}

func (p *Pool) loop(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		j, ok := p.Jobs.DequeueQueued()
		if !ok {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		_ = j
	}
}

func (p *Pool) Wait() { p.wg.Wait() }
