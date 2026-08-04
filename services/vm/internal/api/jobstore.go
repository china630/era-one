package api

import (
	"fmt"
	"sync"
	"time"

	"era/services/vm/internal/models"
	"github.com/google/uuid"
)

// ScanJob — запись о запуске POST /api/v1/vm/scan (in-memory).
type ScanJob struct {
	ID        string           `json:"id"`
	Status    string           `json:"status"`
	Mode      string           `json:"mode"`
	Summary   string           `json:"summary"`
	Note      string           `json:"note,omitempty"`
	Targets   []string         `json:"targets"`
	Findings  []models.Finding `json:"findings,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// JobStore хранит scan jobs и накопленные findings.
type JobStore struct {
	mu       sync.RWMutex
	jobs     map[string]*ScanJob
	order    []string
	findings []models.Finding
}

// NewJobStore создаёт пустое in-memory хранилище.
func NewJobStore() *JobStore {
	return &JobStore{jobs: make(map[string]*ScanJob)}
}

// Record сохраняет завершённый scan job и добавляет findings.
func (s *JobStore) Record(mode, status, summary, note string, targets []string, findings []models.Finding) *ScanJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := &ScanJob{
		ID:        uuid.NewString(),
		Status:    status,
		Mode:      mode,
		Summary:   summary,
		Note:      note,
		Targets:   append([]string(nil), targets...),
		Findings:  append([]models.Finding(nil), findings...),
		CreatedAt: time.Now().UTC(),
	}
	s.jobs[j.ID] = j
	s.order = append(s.order, j.ID)
	s.findings = append(s.findings, findings...)
	return j
}

// List возвращает jobs в порядке создания (новые в конце).
func (s *JobStore) List() []*ScanJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ScanJob, 0, len(s.order))
	for _, id := range s.order {
		if j, ok := s.jobs[id]; ok {
			cp := *j
			cp.Findings = append([]models.Finding(nil), j.Findings...)
			cp.Targets = append([]string(nil), j.Targets...)
			out = append(out, &cp)
		}
	}
	return out
}

// Get возвращает job по id.
func (s *JobStore) Get(id string) (*ScanJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	cp := *j
	cp.Findings = append([]models.Finding(nil), j.Findings...)
	cp.Targets = append([]string(nil), j.Targets...)
	return &cp, true
}

// Findings возвращает накопленные findings из последних сканов.
func (s *JobStore) Findings() []models.Finding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]models.Finding(nil), s.findings...)
}

func summarizeFindings(findings []models.Finding, targets int) string {
	return fmt.Sprintf("targets=%d findings=%d", targets, len(findings))
}
