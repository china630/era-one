package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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
)

type Server struct {
	Ingest *ingestclient.Client
	CMDB   *cmdb.Client
	Gate   *licensegate.Gate
	Tenant string
}

func New(ing *ingestclient.Client, cm *cmdb.Client, gate *licensegate.Gate, tenant string) *Server {
	return &Server{Ingest: ing, CMDB: cm, Gate: gate, Tenant: tenant}
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
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": 1, "node_id": nodeID})
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
	}
	_ = s.Ingest.PostEvents(r.Context(), events)
	writeJSON(w, http.StatusOK, map[string]any{"metrics": m, "events": len(events)})
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
