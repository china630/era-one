package api

import (
	"encoding/json"
	"net/http"
	"os"

	"era/services/vm/internal/cmdb"
	"era/services/vm/internal/cvefeed"
	"era/services/vm/internal/publisher"
	"era/services/vm/internal/scanner"
	"era/services/vm/internal/scheduler"
)

// SetupRoutes регистрирует REST-маршруты модуля /vm.
func SetupRoutes(engine *scanner.Engine, pub *publisher.Publisher, sched *scheduler.Scheduler) *http.ServeMux {
	return SetupRoutesWithJobs(engine, pub, sched, NewJobStore())
}

// SetupRoutesWithJobs — как SetupRoutes, но с явным JobStore (тесты/DI).
func SetupRoutesWithJobs(engine *scanner.Engine, pub *publisher.Publisher, sched *scheduler.Scheduler, jobs *JobStore) *http.ServeMux {
	if jobs == nil {
		jobs = NewJobStore()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/vm/scan", HandleScan(engine, pub, jobs))
	mux.HandleFunc("GET /api/v1/vm/jobs", HandleListJobs(jobs))
	mux.HandleFunc("GET /api/v1/vm/jobs/{id}", HandleGetJob(jobs))
	mux.HandleFunc("GET /api/v1/vm/findings", HandleListFindings(jobs))
	cp := cmdb.NewFromEnv()
	if os.Getenv("ERA_VM_CMDB_SOFTWARE") != "0" {
		mux.HandleFunc("GET /api/v1/vm/software", HandleSoftwareCVE(cp))
		mux.HandleFunc("POST /api/v1/vm/cve-feed/match", HandleCVEFeedMatch(cp, pub))
	}
	if sched != nil {
		mux.HandleFunc("GET /api/v1/scans/schedule", HandleListSchedules(sched))
		mux.HandleFunc("POST /api/v1/scans/schedule", HandleCreateSchedule(sched, engine, pub))
		mux.HandleFunc("PATCH /api/v1/scans/schedule/{id}", HandlePatchSchedule(sched))
		mux.HandleFunc("DELETE /api/v1/scans/schedule/{id}", HandleDeleteSchedule(sched))
	}
	return mux
}

// HandleCVEFeedMatch loads offline CVE feed and matches CMDB software.
func HandleCVEFeedMatch(cp *cmdb.Client, pub *publisher.Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir := os.Getenv("ERA_CVE_FEED_DIR")
		if dir == "" {
			dir = "../../data/cve-feed"
		}
		feed, err := cvefeed.LoadDir(dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var rows []cmdb.SoftwareRow
		if cp != nil {
			rows, err = cp.ListSoftware()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		}
		findings := cvefeed.MatchSoftware(feed, rows)
		if pub != nil && len(findings) > 0 {
			_ = pub.PublishFindings(r.Context(), findings)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"feed_version": feed.Version,
			"cves":         len(feed.CVEs),
			"findings":     findings,
			"count":        len(findings),
		})
	}
}
