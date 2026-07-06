package api

import (
	"encoding/json"
	"net/http"

	"era/services/vm/internal/publisher"
	"era/services/vm/internal/scanner"
	"era/services/vm/internal/scheduler"
)

type scheduleRequest struct {
	Name        string   `json:"name"`
	Targets     []string `json:"targets"`
	CronExpr    string   `json:"cron_expr"`
	Concurrency int      `json:"concurrency"`
}

// HandleListSchedules — GET /api/v1/scans/schedule.
func HandleListSchedules(sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"schedules": sched.List()})
	}
}

// HandleCreateSchedule — POST /api/v1/scans/schedule (immediate run + schedule).
func HandleCreateSchedule(sched *scheduler.Scheduler, engine *scanner.Engine, pub *publisher.Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req scheduleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, "name and targets required", http.StatusBadRequest)
			return
		}
		job := sched.Create(req.Name, req.Targets, req.CronExpr, req.Concurrency)
		status := "scheduled"
		if len(req.Targets) > 0 && engine != nil {
			findings := engine.Run(req.Targets)
			if pub != nil && len(findings) > 0 {
				_ = pub.PublishFindings(r.Context(), findings)
			}
			sched.MarkRun(job.ID, "ok")
			status = "ran"
		}
		writeJSON(w, http.StatusCreated, map[string]any{"job": job, "status": status})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
