package scanner

import (
	"sync/atomic"
	"testing"

	"era/services/vm/internal/models"
)

type countingExecutor struct {
	calls int64
}

func (c *countingExecutor) Execute(target string, tpl *models.Template) ([]models.Finding, error) {
	atomic.AddInt64(&c.calls, 1)
	if tpl == nil {
		return nil, nil
	}
	return []models.Finding{{
		Target:            target,
		TemplateID:        tpl.ID,
		VulnerabilityName: tpl.Info.Name,
	}}, nil
}

func TestEngine_Run_WorkerPool(t *testing.T) {
	ex := &countingExecutor{}
	templates := []*models.Template{
		{ID: "a", Info: models.Info{Name: "n1"}},
		{ID: "b", Info: models.Info{Name: "n2"}},
	}
	targets := []string{"t1", "t2", "t3", "t4", "t5"}

	eng := NewEngine(ex, templates, 3)
	findings := eng.Run(targets)

	wantCalls := int64(len(targets) * len(templates))
	if got := atomic.LoadInt64(&ex.calls); got != wantCalls {
		t.Fatalf("Execute calls: got %d, want %d", got, wantCalls)
	}
	if len(findings) != int(wantCalls) {
		t.Fatalf("findings len: got %d, want %d", len(findings), wantCalls)
	}
}

func TestEngine_Run_NilExecutor(t *testing.T) {
	eng := &Engine{executor: nil, templates: nil, concurrency: 2}
	if got := eng.Run([]string{"x"}); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestEngine_NewEngine_DefaultConcurrency(t *testing.T) {
	ex := &countingExecutor{}
	eng := NewEngine(ex, nil, 0)
	if eng.concurrency != 1 {
		t.Fatalf("concurrency: got %d", eng.concurrency)
	}
}
