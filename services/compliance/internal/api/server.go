package api

import (
	"encoding/json"
	"net/http"
	"time"

	"era/services/compliance/internal/report"
	"era/services/platform/licensegate"
)

type Server struct {
	Gate *licensegate.Gate
	CH   *report.CHClient
}

func New(gate *licensegate.Gate) *Server {
	return &Server{Gate: gate, CH: report.NewCHClientFromEnv()}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/reports/regulatory", s.handleRegulatory)
	mux.HandleFunc("/api/v1/reports/export", s.handleExport)
	mux.HandleFunc("/api/v1/reports/export/zip", s.handleExportZip)
	return mux
}

func (s *Server) handleRegulatory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleNational) {
		http.Error(w, "module national not licensed", http.StatusForbidden)
		return
	}
	tpl := r.URL.Query().Get("template")
	if tpl == "" {
		tpl = "az_cb"
	}
	now := time.Now().UTC()
	start, end := now.Add(-30*24*time.Hour), now
	org := r.URL.Query().Get("org")
	m := report.QueryMetrics(s.CH, org, start, end)
	doc := report.GenerateAZCB(report.Input{
		OrgName: org, PeriodStart: start, PeriodEnd: end,
		TotalEvents: m.TotalEvents, Detections: m.Detections,
		CriticalCount: m.CriticalCount, AssetsCovered: m.AssetsCovered, PIILeaks: m.PIILeaks,
	})
	if tpl != "az_cb" {
		http.Error(w, "unknown template", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleNational) {
		http.Error(w, "module national not licensed", http.StatusForbidden)
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "html"
	}
	now := time.Now().UTC()
	org := r.URL.Query().Get("org")
	start, end := now.Add(-30*24*time.Hour), now
	doc := report.DocumentFromCHWithClient(s.CH, org, start, end)
	switch format {
	case "html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(report.RenderHTML(doc)))
	case "json":
		writeJSON(w, http.StatusOK, doc)
	case "pdf":
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(report.RenderPDF(doc))
	default:
		http.Error(w, "unknown format (html|json|pdf)", http.StatusBadRequest)
	}
}

func (s *Server) handleExportZip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleNational) {
		http.Error(w, "module national not licensed", http.StatusForbidden)
		return
	}
	now := time.Now().UTC()
	org := r.URL.Query().Get("org")
	data, err := report.ExportZIP(org, now.Add(-30*24*time.Hour), now, s.CH)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="era-regulatory-export.zip"`)
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
