// Package chwriter — запись Envelope в ClickHouse (ADR-0007, S1-5).
package chwriter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	erav1 "era/contracts/gen/era/v1"
	"era/services/event-writer/internal/timeline"
	"github.com/oklog/ulid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Writer вставляет события в era_xdr.events.
type Writer struct {
	conn driver.Conn
}

// New подключается к ClickHouse. addr — host:port native (напр. localhost:9000).
func New(addr string) (*Writer, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "era_xdr",
			Username: envOr("ERA_CH_USER", "era"),
			Password: envOr("ERA_CH_PASSWORD", "era_dev_pw"),
		},
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	return &Writer{conn: conn}, nil
}

func (w *Writer) Close() error {
	return w.conn.Close()
}

// Insert записывает событие; для topic xdr.inventory — в inventory_history.
func (w *Writer) Insert(ctx context.Context, topic string, env *erav1.Envelope) error {
	if topic == "xdr.inventory" || isInventoryEnvelope(env) {
		return w.insertInventory(ctx, env)
	}
	row, err := mapEnvelope(env)
	if err != nil {
		return err
	}
	batch, err := w.conn.PrepareBatch(ctx, `INSERT INTO era_xdr.events (
		event_id, correlation_id, schema_version, observed_at, ingested_at,
		tenant_id, environment, cluster_id, node_id, hostname, agent_id, agent_version, platform, src_ip,
		severity, category,
		ocsf_class_uid, ocsf_category_uid, ocsf_activity_id, mitre_tactics, mitre_techniques,
		detection_rule_id, detection_engine, detection_confidence,
		pii_sanitized, payload
	)`)
	if err != nil {
		return err
	}
	if err := batch.AppendStruct(row); err != nil {
		return err
	}
	return batch.Send()
}

func (w *Writer) insertInventory(ctx context.Context, env *erav1.Envelope) error {
	row, err := mapInventoryEnvelope(env)
	if err != nil {
		return err
	}
	batch, err := w.conn.PrepareBatch(ctx, `INSERT INTO era_xdr.inventory_history (
		event_id, tenant_id, node_id, hostname, agent_id, agent_version, platform,
		os_name, os_version, kernel, cpu_cores, ram_mb, software, observed_at, ingested_at
	)`)
	if err != nil {
		return err
	}
	if err := batch.AppendStruct(row); err != nil {
		return err
	}
	return batch.Send()
}

// QueryRecent возвращает последние события (S1-10 API).
func (w *Writer) QueryRecent(ctx context.Context, nodeID string, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	q := `SELECT event_id, tenant_id, node_id, hostname, observed_at, ingested_at,
		severity, category, pii_sanitized, payload
		FROM era_xdr.events`
	args := []any{}
	if nodeID != "" {
		q += ` WHERE node_id = ?`
		args = append(args, nodeID)
	}
	q += ` ORDER BY observed_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := w.conn.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var (
			eventID, tenantID, node, host, severity, category, payload string
			observedAt, ingestedAt                                     time.Time
			pii                                                        uint8
		)
		if err := rows.Scan(&eventID, &tenantID, &node, &host, &observedAt, &ingestedAt,
			&severity, &category, &pii, &payload); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"event_id":      eventID,
			"tenant_id":     tenantID,
			"node_id":       node,
			"hostname":      host,
			"observed_at":   observedAt.UTC().Format(time.RFC3339Nano),
			"ingested_at":   ingestedAt.UTC().Format(time.RFC3339Nano),
			"severity":      severity,
			"category":      category,
			"pii_sanitized": pii,
			"payload":       payload,
		})
	}
	return out, rows.Err()
}

// QueryTimeline возвращает merged timeline по node_id и/или correlation_id (ADR-0017).
func (w *Writer) QueryTimeline(ctx context.Context, nodeID, correlationID string, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var events, dets []timeline.Entry

	evQ := `SELECT event_id, correlation_id, node_id, observed_at, severity, category,
		detection_engine, payload FROM era_xdr.events WHERE 1=1`
	var evArgs []any
	if nodeID != "" {
		evQ += ` AND node_id = ?`
		evArgs = append(evArgs, nodeID)
	}
	if correlationID != "" {
		evQ += ` AND correlation_id = ?`
		evArgs = append(evArgs, correlationID)
	}
	evQ += ` ORDER BY observed_at DESC LIMIT ?`
	evArgs = append(evArgs, limit)

	rows, err := w.conn.Query(ctx, evQ, evArgs...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var eid, cid, node, sev, cat, engine, payload string
			var at time.Time
			if rows.Scan(&eid, &cid, &node, &at, &sev, &cat, &engine, &payload) == nil {
				src := "agent"
				if engine != "" {
					src = engine
				}
				events = append(events, timeline.Entry{
					Kind: "event", At: at.UTC().Format(time.RFC3339Nano), NodeID: node,
					Severity: sev, Category: cat, Source: src, Summary: cat + " event",
					EventID: eid, CorrelationID: cid, Payload: payload,
				})
			}
		}
	}

	detQ := `SELECT detection_id, event_id, correlation_id, node_id, observed_at, severity, engine, rule_name
		FROM era_xdr.detections WHERE 1=1`
	var detArgs []any
	if nodeID != "" {
		detQ += ` AND node_id = ?`
		detArgs = append(detArgs, nodeID)
	}
	if correlationID != "" {
		detQ += ` AND correlation_id = ?`
		detArgs = append(detArgs, correlationID)
	}
	detQ += ` ORDER BY observed_at DESC LIMIT ?`
	detArgs = append(detArgs, limit)

	drows, err := w.conn.Query(ctx, detQ, detArgs...)
	if err == nil {
		defer drows.Close()
		for drows.Next() {
			var detID, eid, cid, node, sev, engine, rule string
			var at time.Time
			if drows.Scan(&detID, &eid, &cid, &node, &at, &sev, &engine, &rule) == nil {
				dets = append(dets, timeline.Entry{
					Kind: "detection", At: at.UTC().Format(time.RFC3339Nano), NodeID: node,
					Severity: sev, Source: engine, Summary: rule,
					DetectionID: detID, EventID: eid, CorrelationID: cid,
				})
			}
		}
	}

	merged := timeline.Merge(events, dets)
	out := make([]map[string]any, len(merged))
	for i, e := range merged {
		out[i] = entryToMap(e)
	}
	return out, nil
}

func entryToMap(e timeline.Entry) map[string]any {
	m := map[string]any{
		"kind": e.Kind, "at": e.At, "node_id": e.NodeID, "severity": e.Severity,
		"source": e.Source, "summary": e.Summary,
	}
	if e.Category != "" {
		m["category"] = e.Category
	}
	if e.EventID != "" {
		m["event_id"] = e.EventID
	}
	if e.DetectionID != "" {
		m["detection_id"] = e.DetectionID
	}
	if e.CorrelationID != "" {
		m["correlation_id"] = e.CorrelationID
	}
	if e.Payload != "" {
		m["payload"] = e.Payload
	}
	return m
}

type eventRow struct {
	EventID             string    `ch:"event_id"`
	CorrelationID       string    `ch:"correlation_id"`
	SchemaVersion       string    `ch:"schema_version"`
	ObservedAt          time.Time `ch:"observed_at"`
	IngestedAt          time.Time `ch:"ingested_at"`
	TenantID            string    `ch:"tenant_id"`
	Environment         string    `ch:"environment"`
	ClusterID           string    `ch:"cluster_id"`
	NodeID              string    `ch:"node_id"`
	Hostname            string    `ch:"hostname"`
	AgentID             string    `ch:"agent_id"`
	AgentVersion        string    `ch:"agent_version"`
	Platform            string    `ch:"platform"`
	SrcIP               []string  `ch:"src_ip"`
	Severity            string    `ch:"severity"`
	Category            string    `ch:"category"`
	OcsfClassUID        uint32    `ch:"ocsf_class_uid"`
	OcsfCategoryUID     uint32    `ch:"ocsf_category_uid"`
	OcsfActivityID      uint32    `ch:"ocsf_activity_id"`
	MitreTactics        []string  `ch:"mitre_tactics"`
	MitreTechniques     []string  `ch:"mitre_techniques"`
	DetectionRuleID     string    `ch:"detection_rule_id"`
	DetectionEngine     string    `ch:"detection_engine"`
	DetectionConfidence float32   `ch:"detection_confidence"`
	PiiSanitized        uint8     `ch:"pii_sanitized"`
	Payload             string    `ch:"payload"`
}

func mapEnvelope(env *erav1.Envelope) (*eventRow, error) {
	if env == nil {
		return nil, fmt.Errorf("nil envelope")
	}
	eid, err := ulidFromBytes(env.GetEventId())
	if err != nil {
		return nil, err
	}
	cid := ""
	if b := env.GetCorrelationId(); len(b) == 16 {
		if s, err := ulidFromBytes(b); err == nil {
			cid = s
		}
	}
	src := env.GetSource()
	payloadJSON, _ := json.Marshal(payloadBody(env))

	obs := ts(env.GetObservedAt())
	ing := ts(env.GetIngestedAt())
	if ing.IsZero() {
		ing = time.Now().UTC()
	}

	row := &eventRow{
		EventID:       eid,
		CorrelationID: cid,
		SchemaVersion: env.GetSchemaVersion(),
		ObservedAt:    obs,
		IngestedAt:    ing,
		TenantID:      src.GetTenantId(),
		Environment:   src.GetEnvironment(),
		ClusterID:     src.GetClusterId(),
		NodeID:        src.GetNodeId(),
		Hostname:      src.GetHostname(),
		AgentID:       src.GetAgentId(),
		AgentVersion:  src.GetAgentVersion(),
		Platform:      platformStr(src.GetPlatform()),
		SrcIP:         src.GetIp(),
		Severity:      severityStr(env.GetSeverity()),
		Category:      categoryStr(env.GetCategory()),
		PiiSanitized:  boolToU8(env.GetPiiSanitized()),
		Payload:       string(payloadJSON),
	}
	if o := env.GetOcsf(); o != nil {
		row.OcsfClassUID = o.GetClassUid()
		row.OcsfCategoryUID = o.GetCategoryUid()
		row.OcsfActivityID = o.GetActivityId()
	}
	if m := env.GetMitre(); m != nil {
		row.MitreTactics = m.GetTacticIds()
		row.MitreTechniques = m.GetTechniqueIds()
	}
	if d := env.GetDetection(); d != nil {
		row.DetectionRuleID = d.GetRuleId()
		row.DetectionEngine = d.GetEngine()
		row.DetectionConfidence = d.GetConfidence()
	}
	return row, nil
}

func payloadBody(env *erav1.Envelope) any {
	switch p := env.GetPayload().(type) {
	case *erav1.Envelope_Process:
		return p.Process
	case *erav1.Envelope_Network:
		return p.Network
	case *erav1.Envelope_File:
		return p.File
	case *erav1.Envelope_Auth:
		return p.Auth
	case *erav1.Envelope_Raw:
		return p.Raw
	default:
		return map[string]string{"type": "unknown"}
	}
}

func ulidFromBytes(b []byte) (string, error) {
	if len(b) != 16 {
		return "", fmt.Errorf("event_id must be 16 bytes ULID, got %d", len(b))
	}
	var id ulid.ULID
	copy(id[:], b)
	return id.String(), nil
}

func ts(t *timestamppb.Timestamp) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.AsTime().UTC()
}

func platformStr(p erav1.Platform) string {
	switch p {
	case erav1.Platform_PLATFORM_WINDOWS:
		return "windows"
	case erav1.Platform_PLATFORM_LINUX:
		return "linux"
	case erav1.Platform_PLATFORM_MACOS:
		return "macos"
	default:
		return "unspecified"
	}
}

func severityStr(s erav1.Severity) string {
	switch s {
	case erav1.Severity_SEVERITY_INFO:
		return "info"
	case erav1.Severity_SEVERITY_LOW:
		return "low"
	case erav1.Severity_SEVERITY_MEDIUM:
		return "medium"
	case erav1.Severity_SEVERITY_HIGH:
		return "high"
	case erav1.Severity_SEVERITY_CRITICAL:
		return "critical"
	default:
		return "unspecified"
	}
}

func categoryStr(c erav1.EventCategory) string {
	switch c {
	case erav1.EventCategory_EVENT_CATEGORY_PROCESS:
		return "process"
	case erav1.EventCategory_EVENT_CATEGORY_NETWORK:
		return "network"
	case erav1.EventCategory_EVENT_CATEGORY_FILE:
		return "file"
	case erav1.EventCategory_EVENT_CATEGORY_REGISTRY:
		return "registry"
	case erav1.EventCategory_EVENT_CATEGORY_AUTH:
		return "auth"
	case erav1.EventCategory_EVENT_CATEGORY_DNS:
		return "dns"
	case erav1.EventCategory_EVENT_CATEGORY_MODULE:
		return "module"
	default:
		return "unspecified"
	}
}

func boolToU8(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return strings.TrimSpace(v)
	}
	return def
}
