package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"era/services/vm/internal/models"
	"era/services/vm/internal/publisher"
	"era/services/vm/internal/scanner"
)

// ScanRequest описывает входной JSON-запрос на запуск сканирования.
type ScanRequest struct {
	Targets     []string `json:"targets"`
	Concurrency int      `json:"concurrency"`
	// Mode: ""/"http" — шаблонный HTTP-скан; "credentialed"/"credentialed_stub" — SSH banner stub.
	Mode string `json:"mode"`
}

// ScanResponse описывает JSON-ответ с результатами сканирования.
type ScanResponse struct {
	Status   string           `json:"status"`
	JobID    string           `json:"job_id"`
	Mode     string           `json:"mode"`
	Summary  string           `json:"summary"`
	Note     string           `json:"note,omitempty"`
	Findings []models.Finding `json:"findings"`
}

const credentialedStubNote = "credentialed_stub: SSH banner grab only — not full authenticated vuln scan"

func normalizeScanMode(mode string) (normalized, note string) {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "credentialed", "credentialed_stub", "ssh_stub":
		return "credentialed_stub", credentialedStubNote
	case "", "http":
		return "http", ""
	default:
		return m, ""
	}
}

// HandleScan возвращает HTTP-хэндлер запуска сканирования.
func HandleScan(engine *scanner.Engine, pub *publisher.Publisher, store *JobStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		mode, note := normalizeScanMode(req.Mode)
		var findings []models.Finding
		if mode == "credentialed_stub" {
			ex := scanner.NewSSHCredentialed()
			tpl := &models.Template{
				ID:   "ssh-banner-stub",
				Info: models.Info{Name: "SSH banner (credentialed stub)", Severity: "info"},
			}
			for _, t := range req.Targets {
				fs, err := ex.Execute(t, tpl)
				if err != nil {
					log.Printf("credentialed stub %s: %v", t, err)
					continue
				}
				findings = append(findings, fs...)
			}
		} else {
			findings = engine.Run(req.Targets)
		}

		if pub != nil && len(findings) > 0 {
			if err := pub.PublishFindings(r.Context(), findings); err != nil {
				log.Printf("kafka publish findings: %v", err)
			}
		}

		summary := summarizeFindings(findings, len(req.Targets))
		jobID := ""
		if store != nil {
			job := store.Record(mode, "ok", summary, note, req.Targets, findings)
			jobID = job.ID
		}

		resp := ScanResponse{
			Status:   "ok",
			JobID:    jobID,
			Mode:     mode,
			Summary:  summary,
			Note:     note,
			Findings: findings,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// HandleListJobs — GET /api/v1/vm/jobs.
func HandleListJobs(store *JobStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jobs := []*ScanJob{}
		if store != nil {
			jobs = store.List()
		}
		writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
	}
}

// HandleGetJob — GET /api/v1/vm/jobs/{id}.
func HandleGetJob(store *JobStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.PathValue("id")
		if id == "" || store == nil {
			http.NotFound(w, r)
			return
		}
		job, ok := store.Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, job)
	}
}

// HandleListFindings — GET /api/v1/vm/findings.
func HandleListFindings(store *JobStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		findings := []models.Finding{}
		if store != nil {
			findings = store.Findings()
		}
		writeJSON(w, http.StatusOK, map[string]any{"findings": findings, "count": len(findings)})
	}
}
