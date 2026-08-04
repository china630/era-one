package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"era/services/observe/internal/adapters"
	"era/services/observe/internal/cmdb"
	"era/services/observe/internal/discovery"
	"era/services/observe/internal/envelope"
	ingestclient "era/services/observe/internal/ingest"
	"era/services/observe/internal/netflow"
	"era/services/observe/internal/snmp"
	"era/services/platform/licensegate"
	"era/services/platform/metrics"
	erav1 "era/contracts/gen/era/v1"
	"github.com/google/uuid"
)

// Alert — in-memory lab alert (не замена detection-engine).
type Alert struct {
	ID        string    `json:"id"`
	NodeID    string    `json:"node_id"`
	Severity  string    `json:"severity"`
	Summary   string    `json:"summary"`
	Source    string    `json:"source"`
	Acked     bool      `json:"acked"`
	AckedBy   string    `json:"acked_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// PollerSchedule — простой CRUD расписания SNMP poll targets.
type PollerSchedule struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Targets    []string  `json:"targets"`
	IntervalSec int      `json:"interval_sec"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

type Server struct {
	Ingest *ingestclient.Client
	CMDB   *cmdb.Client
	Gate   *licensegate.Gate
	Tenant string

	mu       sync.RWMutex
	alerts   []*Alert
	pollers  []*PollerSchedule
}

func New(ing *ingestclient.Client, cm *cmdb.Client, gate *licensegate.Gate, tenant string) *Server {
	now := time.Now().UTC()
	return &Server{
		Ingest: ing, CMDB: cm, Gate: gate, Tenant: tenant,
		alerts: []*Alert{
			{
				ID: "alert-lab-1", NodeID: "net-10-0-0-1", Severity: "warning",
				Summary: "high egress on Gi0/1 (sim)", Source: "observe_snmp",
				CreatedAt: now,
			},
			{
				ID: "alert-lab-2", NodeID: "sw-01", Severity: "info",
				Summary: "lab seed alert", Source: "lab",
				CreatedAt: now,
			},
		},
		pollers: []*PollerSchedule{
			{
				ID: "poller-default", Name: "default-sim",
				Targets: []string{"10.0.0.1"}, IntervalSec: 60, Enabled: true,
				CreatedAt: now,
			},
		},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/api/v1/webhooks/prtg", s.handlePRTG)
	mux.HandleFunc("/api/v1/webhooks/zabbix", s.handleZabbix)
	mux.HandleFunc("/api/v1/webhooks/syslog", s.handleSyslog)
	mux.HandleFunc("/api/v1/snmp/poll", s.handleSNMPPoll)
	mux.HandleFunc("/api/v1/discovery/sweep", s.handleDiscovery)
	mux.HandleFunc("/api/v1/netflow/line", s.handleNetflow)
	mux.HandleFunc("/api/v1/devices", s.handleDevices)
	mux.HandleFunc("/api/v1/devices/", s.handleDeviceDetail)
	mux.HandleFunc("/api/v1/alerts", s.handleAlerts)
	mux.HandleFunc("/api/v1/alerts/", s.handleAlertSub)
	mux.HandleFunc("/api/v1/pollers", s.handlePollers)
	mux.HandleFunc("/api/v1/pollers/", s.handlePollerDetail)
	mux.HandleFunc("/api/v1/topology", s.handleTopology)
	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(uiDir()))))
	return mux
}

func uiDir() string {
	if d := os.Getenv("ERA_OBSERVE_UI_DIR"); d != "" {
		return d
	}
	return "ui/observe"
}

func (s *Server) requireObserve(w http.ResponseWriter) bool {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleObserve) {
		http.Error(w, "observe module not licensed", http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handlePRTG(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requireObserve(w) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	body, _ := io.ReadAll(r.Body)
	wHook, err := adapters.ParsePRTG(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.emitNMS(w, r, wHook.NodeID(), "prtg", wHook.Summary(), wHook.Detail())
}

func (s *Server) handleZabbix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requireObserve(w) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	body, _ := io.ReadAll(r.Body)
	wHook, err := adapters.ParseZabbix(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.emitNMS(w, r, wHook.NodeID(), "zabbix", wHook.Summary(), wHook.Trigger)
}

func (s *Server) handleSyslog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requireObserve(w) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	body, _ := io.ReadAll(r.Body)
	host, summary, err := adapters.ParseSyslogNetwork(strings.TrimSpace(string(body)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node := "net-" + strings.ReplaceAll(host, ".", "-")
	s.emitNMS(w, r, node, "syslog", summary, host)
}

func (s *Server) emitNMS(w http.ResponseWriter, r *http.Request, nodeID, source, summary, detail string) {
	env := envelope.FromNMSAlert(s.tenant(), nodeID, source, summary, detail)
	_ = s.Ingest.PostEvents(r.Context(), []*erav1.Envelope{env})
	_, _ = s.CMDB.ReconcileNetwork(r.Context(), cmdb.NetworkAsset{
		NodeID: nodeID, TenantID: s.tenant(), Hostname: nodeID, IPAddrs: ipFromNode(nodeID),
	})
	s.pushAlert(nodeID, "warning", summary, source)
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": 1, "node_id": nodeID})
}

func (s *Server) pushAlert(nodeID, severity, summary, source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, &Alert{
		ID: uuid.NewString(), NodeID: nodeID, Severity: severity,
		Summary: summary, Source: source, CreatedAt: time.Now().UTC(),
	})
}

func (s *Server) handleSNMPPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requireObserve(w) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	target := r.URL.Query().Get("target")
	if target == "" {
		target = "10.0.0.1"
	}
	m := snmp.Poll(target)
	var events []*erav1.Envelope
	if ok, msg := snmp.HighEgressAlert(m); ok {
		node := "net-" + strings.ReplaceAll(target, ".", "-")
		events = append(events, envelope.FromNMSAlert(s.tenant(), node, "observe_snmp", msg, target))
		_, _ = s.CMDB.ReconcileNetwork(r.Context(), cmdb.NetworkAsset{
			NodeID: node, TenantID: s.tenant(), Hostname: target, IPAddrs: []string{target},
		})
		s.pushAlert(node, "warning", msg, "observe_snmp")
	}
	_ = s.Ingest.PostEvents(r.Context(), events)
	// Honest label: sim path always reports metrics_source=sim (see snmp.PollSimulated).
	writeJSON(w, http.StatusOK, map[string]any{"metrics": m, "events": len(events), "metrics_source": m.MetricsSource})
}

func (s *Server) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requireObserve(w) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	cidr := r.URL.Query().Get("cidr")
	nodes := discovery.Sweep(cidr)
	var registered int
	for _, n := range nodes {
		nodeID := "net-" + strings.ReplaceAll(n.IP, ".", "-")
		res, err := s.CMDB.ReconcileNetwork(r.Context(), cmdb.NetworkAsset{
			NodeID: nodeID, TenantID: s.tenant(), Hostname: n.Hostname, Kind: n.Kind,
			IPAddrs: []string{n.IP}, MACAddrs: []string{n.MAC},
		})
		if err != nil {
			log.Printf("observe discovery cmdb: %v", err)
			continue
		}
		if !res.Conflict {
			registered++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"discovered": len(nodes), "registered": registered, "nodes": nodes})
}

func (s *Server) handleNetflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !s.requireObserve(w) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	body, _ := io.ReadAll(r.Body)
	line := strings.TrimSpace(string(body))
	var rec netflow.Record
	if strings.Contains(line, ",") {
		var err error
		rec, err = netflow.ParseLine(line)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		_, recs, err := netflow.ParseV5(body)
		if err != nil || len(recs) == 0 {
			http.Error(w, "netflow parse failed", http.StatusBadRequest)
			return
		}
		rec = recs[0]
	}
	node := "net-" + strings.ReplaceAll(rec.SrcIP, ".", "-")
	detail := rec.DstIP + ":" + strconv.FormatUint(uint64(rec.DstPort), 10)
	env := envelope.FromNMSAlert(s.tenant(), node, "netflow", "flow "+rec.Proto, detail)
	_ = s.Ingest.PostEvents(r.Context(), []*erav1.Envelope{env})
	writeJSON(w, http.StatusOK, map[string]any{"record": rec})
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || !s.requireObserve(w) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	assets, err := s.CMDB.ListNetwork(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": assets})
}

func (s *Server) handleDeviceDetail(w http.ResponseWriter, r *http.Request) {
	if !s.requireObserve(w) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/devices/"), "/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	asset, err := s.CMDB.GetNetwork(r.Context(), id)
	if err != nil || asset == nil {
		http.NotFound(w, r)
		return
	}
	// Attach last sim metrics when available (honest metrics_source).
	var metrics any
	if len(asset.IPAddrs) > 0 {
		m := snmp.PollSimulated(asset.IPAddrs[0])
		metrics = m
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"device":         asset,
		"metrics":        metrics,
		"metrics_source": "sim",
	})
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if !s.requireObserve(w) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Alert, len(s.alerts))
	copy(out, s.alerts)
	writeJSON(w, http.StatusOK, map[string]any{"alerts": out})
}

func (s *Server) handleAlertSub(w http.ResponseWriter, r *http.Request) {
	if !s.requireObserve(w) {
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	id := parts[0]
	if len(parts) == 2 && parts[1] == "ack" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			By string `json:"by"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.By == "" {
			body.By = "operator"
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, a := range s.alerts {
			if a.ID == id {
				a.Acked = true
				a.AckedBy = body.By
				writeJSON(w, http.StatusOK, a)
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handlePollers(w http.ResponseWriter, r *http.Request) {
	if !s.requireObserve(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		defer s.mu.RUnlock()
		out := make([]*PollerSchedule, len(s.pollers))
		copy(out, s.pollers)
		writeJSON(w, http.StatusOK, map[string]any{"pollers": out})
	case http.MethodPost:
		var body PollerSchedule
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		if body.ID == "" {
			body.ID = "poller-" + uuid.NewString()[:8]
		}
		if body.IntervalSec <= 0 {
			body.IntervalSec = 60
		}
		if body.Targets == nil {
			body.Targets = []string{}
		}
		body.CreatedAt = time.Now().UTC()
		s.mu.Lock()
		s.pollers = append(s.pollers, &body)
		s.mu.Unlock()
		writeJSON(w, http.StatusCreated, body)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePollerDetail(w http.ResponseWriter, r *http.Request) {
	if !s.requireObserve(w) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/pollers/"), "/")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, p := range s.pollers {
			if p.ID == id {
				writeJSON(w, http.StatusOK, p)
				return
			}
		}
		http.NotFound(w, r)
	case http.MethodDelete:
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, p := range s.pollers {
			if p.ID == id {
				s.pollers = append(s.pollers[:i], s.pollers[i+1:]...)
				writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
				return
			}
		}
		http.NotFound(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || !s.requireObserve(w) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	assets, err := s.CMDB.ListNetwork(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	nodes := make([]map[string]string, 0, len(assets))
	edges := make([]map[string]string, 0)
	for _, a := range assets {
		nodes = append(nodes, map[string]string{
			"id": a.NodeID, "label": a.Hostname, "kind": firstNonEmpty(a.AssetKind, a.Kind),
		})
		if a.Managed {
			edges = append(edges, map[string]string{"from": "ingest", "to": a.NodeID, "type": "telemetry"})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes, "edges": edges})
}

func (s *Server) tenant() string {
	if s.Tenant != "" {
		return s.Tenant
	}
	return "default"
}

func ipFromNode(nodeID string) []string {
	if strings.HasPrefix(nodeID, "net-") {
		ip := strings.TrimPrefix(nodeID, "net-")
		ip = strings.ReplaceAll(ip, "-", ".")
		return []string{ip}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
