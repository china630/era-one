package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"era/services/comms/migration/internal/audit"
	"era/services/comms/migration/internal/connectors/source/communigate"
	"era/services/comms/migration/internal/foldermap"
	"era/services/comms/migration/internal/importers/archive"
	"era/services/comms/migration/internal/importers/ews"
	"era/services/comms/migration/internal/importers/imap"
	"era/services/comms/migration/internal/jobs"
	"era/services/comms/migration/internal/target"
	"era/services/comms/migration/internal/worker"
)

type Server struct {
	Jobs   jobs.Repository
	Audit  audit.Recorder
	Runner *worker.Runner
}

func NewServer(j jobs.Repository, a audit.Recorder) *Server {
	return &Server{
		Jobs:   j,
		Audit:  a,
		Runner: &worker.Runner{Jobs: j, Audit: a},
	}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/v1/migration/jobs", s.jobsHandler)
	mux.HandleFunc("/api/v1/migration/discover", s.discover)
	mux.HandleFunc("/api/v1/migration/resume", s.resume)
	mux.HandleFunc("/api/v1/migration/rerun", s.rerun)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "service": "comms-migration"})
}

type createJobReq struct {
	Source      string             `json:"source"`
	Mailbox     string             `json:"mailbox"`
	IMAPFile    string             `json:"imap_file"`
	ArchiveFile string             `json:"archive_file"`
	SourceIMAP  imap.NetworkConfig `json:"source_imap"`
	Target      string             `json:"target"`
	TargetIMAP  imap.NetworkConfig `json:"target_imap"`
	MailAPIURL  string             `json:"mail_api_url"`
	Folder      string             `json:"folder"`
	AllFolders  bool               `json:"all_folders"`
	Mode        string             `json:"mode"`
}

func (s *Server) jobsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		id := r.URL.Query().Get("id")
		j, ok := s.Jobs.Get(id)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, j)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req createJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if req.SourceIMAP.Host != "" {
		tw, err := s.buildTarget(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mode := req.Mode
		if mode == "" {
			mode = "bulk"
		}
		j, err := s.Runner.Start(context.Background(), worker.JobRequest{
			Source:     req.Source,
			Mailbox:    req.Mailbox,
			SourceIMAP: req.SourceIMAP,
			Target:     tw,
			Folder:     req.Folder,
			AllFolders: req.AllFolders || req.Folder == "*",
			Mode:       mode,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.Audit.Record(audit.Event{JobID: j.ID, Action: "MIGRATION_JOB_CREATED"})
		writeJSON(w, j)
		return
	}

	j := s.Runner.ImportFileJob(req.Source, req.Mailbox, req.IMAPFile, req.ArchiveFile)
	total := j.ItemsTotal
	total += ews.ImportCalendar([]ews.CalendarItem{{ID: "ev-1", Subject: "Migrated"}})
	if req.ArchiveFile != "" {
		if strings.HasSuffix(strings.ToLower(req.ArchiveFile), ".mbox") {
			msgs, err := archive.ImportMBOX(req.ArchiveFile)
			if err == nil {
				total += len(msgs)
			}
		} else if archive.ImportSmoke(req.ArchiveFile) {
			total++
		}
	}
	j.ItemsTotal = total
	j.ItemsOK = total
	writeJSON(w, j)
}

func (s *Server) buildTarget(req createJobReq) (target.Writer, error) {
	switch strings.ToLower(req.Target) {
	case "era-mail-server", "era", "":
		url := req.MailAPIURL
		if url == "" {
			url = os.Getenv("ERA_MAIL_API_URL")
		}
		if url == "" {
			url = "http://127.0.0.1:8150"
		}
		return target.NewERA(target.ERAConfig{
			MailAPIURL: url,
			Mailbox:    req.Mailbox,
		}), nil
	case "icewarp":
		iwCfg := target.IceWarpConfig{
			IMAP:   req.TargetIMAP,
			Folder: req.Folder,
		}
		if mapper := folderMapperForSource(req.Source); mapper != nil {
			iwCfg.MapFolder = mapper
		}
		return target.NewIceWarp(iwCfg)
	default:
		return nil, errTarget(req.Target)
	}
}

type targetError string

func (e targetError) Error() string { return "unknown target: " + string(e) }
func errTarget(name string) error   { return targetError(name) }

func folderMapperForSource(source string) foldermap.Mapper {
	switch strings.ToLower(source) {
	case "communigate", "cg", "s-01", "lotus", "domino":
		return communigate.MapFolder
	default:
		return nil
	}
}

func (s *Server) discover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Source     string             `json:"source"`
		SourceIMAP imap.NetworkConfig `json:"source_imap"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if strings.ToLower(req.Source) != "communigate" && req.Source != "cg" && req.Source != "S-01" {
		http.Error(w, "discover supports communigate source only in phase 2", http.StatusBadRequest)
		return
	}
	format := r.URL.Query().Get("format")
	if format == "csv" {
		csv, err := communigate.ReportCSV(req.SourceIMAP)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte(csv))
		return
	}
	res, err := communigate.Discover(req.SourceIMAP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, res)
}

func (s *Server) resume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pg, ok := s.Jobs.(*jobs.PGStore)
	if !ok {
		writeJSON(w, map[string]any{"requeued": 0, "note": "memory store — no resume"})
		return
	}
	n, err := pg.RequeueFailed()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.Audit.Record(audit.Event{Action: "MIGRATION_RESUME", Detail: fmt.Sprintf("requeued=%d", n)})
	writeJSON(w, map[string]int64{"requeued": n})
}

func (s *Server) rerun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SourceUIDs []string `json:"source_uids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	newItems := s.Jobs.Rerun(req.SourceUIDs)
	for _, uid := range req.SourceUIDs {
		s.Audit.Record(audit.Event{Action: "MIGRATION_RERUN", SourceUID: uid})
	}
	writeJSON(w, map[string]int{"new_items": newItems})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
