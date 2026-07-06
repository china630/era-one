package api

import (
	"net/http"
	"os"

	"era/services/vm/internal/cmdb"
	"era/services/vm/internal/publisher"
	"era/services/vm/internal/scanner"
	"era/services/vm/internal/scheduler"
)

// SetupRoutes регистрирует REST-маршруты модуля /vm.
func SetupRoutes(engine *scanner.Engine, pub *publisher.Publisher, sched *scheduler.Scheduler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/vm/scan", HandleScan(engine, pub))
	cp := cmdb.NewFromEnv()
	if os.Getenv("ERA_VM_CMDB_SOFTWARE") != "0" {
		mux.HandleFunc("GET /api/v1/vm/software", HandleSoftwareCVE(cp))
	}
	if sched != nil {
		mux.HandleFunc("GET /api/v1/scans/schedule", HandleListSchedules(sched))
		mux.HandleFunc("POST /api/v1/scans/schedule", HandleCreateSchedule(sched, engine, pub))
	}
	return mux
}
