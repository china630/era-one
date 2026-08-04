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
	Enabled     *bool    `json:"enabled,omitempty"`
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

// HandlePatchSchedule — PATCH /api/v1/scans/schedule/{id}.
func HandlePatchSchedule(sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.PathValue("id")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		var req scheduleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		job, ok := sched.Update(id, scheduler.UpdateFields{
			Name:        req.Name,
			Targets:     req.Targets,
			CronExpr:    req.CronExpr,
			Concurrency: req.Concurrency,
			Enabled:     req.Enabled,
		})
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"job": job})
	}
}

// HandleDeleteSchedule — DELETE /api/v1/scans/schedule/{id}.
func HandleDeleteSchedule(sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.PathValue("id")
		if id == "" || !sched.Delete(id) {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
