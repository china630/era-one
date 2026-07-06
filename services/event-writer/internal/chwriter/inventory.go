package chwriter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"github.com/oklog/ulid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type inventoryRow struct {
	EventID       string    `ch:"event_id"`
	TenantID      string    `ch:"tenant_id"`
	NodeID        string    `ch:"node_id"`
	Hostname      string    `ch:"hostname"`
	AgentID       string    `ch:"agent_id"`
	AgentVersion  string    `ch:"agent_version"`
	Platform      string    `ch:"platform"`
	OSName        string    `ch:"os_name"`
	OSVersion     string    `ch:"os_version"`
	Kernel        string    `ch:"kernel"`
	CPUCores      uint32    `ch:"cpu_cores"`
	RAMMB         uint64    `ch:"ram_mb"`
	Software      string    `ch:"software"`
	ObservedAt    time.Time `ch:"observed_at"`
	IngestedAt    time.Time `ch:"ingested_at"`
}

func isInventoryEnvelope(env *erav1.Envelope) bool {
	if env == nil {
		return false
	}
	raw := env.GetRaw()
	return raw != nil && strings.HasPrefix(raw.GetSourceType(), "era.inventory")
}

func mapInventoryEnvelope(env *erav1.Envelope) (*inventoryRow, error) {
	if !isInventoryEnvelope(env) {
		return nil, fmt.Errorf("not an inventory envelope")
	}
	eid, err := ulidFromBytes(env.GetEventId())
	if err != nil {
		return nil, err
	}
	src := env.GetSource()
	fields := env.GetRaw().GetFields().GetFields()

	swJSON := ""
	if sw := invStrField(fields, "software"); sw != "" {
		swJSON = sw
	} else if items := invSoftwareFromFields(fields); len(items) > 0 {
		b, _ := json.Marshal(items)
		swJSON = string(b)
	}

	obs := ts(env.GetObservedAt())
	ing := ts(env.GetIngestedAt())
	if ing.IsZero() {
		ing = time.Now().UTC()
	}
	if obs.IsZero() {
		obs = ing
	}

	hostname := invStrField(fields, "hostname")
	if hostname == "" {
		hostname = src.GetHostname()
	}
	platform := invStrField(fields, "platform")
	if platform == "" {
		platform = platformStr(src.GetPlatform())
	}

	return &inventoryRow{
		EventID:      eid,
		TenantID:     src.GetTenantId(),
		NodeID:       src.GetNodeId(),
		Hostname:     hostname,
		AgentID:      src.GetAgentId(),
		AgentVersion: src.GetAgentVersion(),
		Platform:     platform,
		OSName:       invStrField(fields, "os_name"),
		OSVersion:    invStrField(fields, "os_version"),
		Kernel:       invStrField(fields, "kernel"),
		CPUCores:     uint32(invNumField(fields, "cpu_count")),
		RAMMB:        uint64(invNumField(fields, "total_memory_mb")),
		Software:     swJSON,
		ObservedAt:   obs,
		IngestedAt:   ing,
	}, nil
}

func invStrField(fields map[string]*structpb.Value, key string) string {
	if fields == nil {
		return ""
	}
	v, ok := fields[key]
	if !ok || v == nil {
		return ""
	}
	return v.GetStringValue()
}

func invNumField(fields map[string]*structpb.Value, key string) float64 {
	if fields == nil {
		return 0
	}
	v, ok := fields[key]
	if !ok || v == nil {
		return 0
	}
	return v.GetNumberValue()
}

type softwareItem struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Vendor  string `json:"vendor,omitempty"`
	Source  string `json:"source,omitempty"`
}

func invSoftwareFromFields(fields map[string]*structpb.Value) []softwareItem {
	// fallback: individual software fields not used in current agent path
	return nil
}

// inventoryRowJSON — стабильная сериализация для golden-теста.
func inventoryRowJSON(row *inventoryRow) ([]byte, error) {
	if row == nil {
		return nil, fmt.Errorf("nil row")
	}
	m := map[string]any{
		"event_id":       row.EventID,
		"tenant_id":      row.TenantID,
		"node_id":        row.NodeID,
		"hostname":       row.Hostname,
		"agent_id":       row.AgentID,
		"agent_version":  row.AgentVersion,
		"platform":       row.Platform,
		"os_name":        row.OSName,
		"os_version":     row.OSVersion,
		"kernel":         row.Kernel,
		"cpu_cores":      row.CPUCores,
		"ram_mb":         row.RAMMB,
		"software":       row.Software,
		"observed_at":    row.ObservedAt.UTC().Format(time.RFC3339Nano),
		"ingested_at":    row.IngestedAt.UTC().Format(time.RFC3339Nano),
	}
	return json.Marshal(m)
}

// testInventoryEnvelope builds a minimal inventory envelope for unit tests.
func testInventoryEnvelope() *erav1.Envelope {
	obs := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sw := `[{"name":"curl","version":"8.5.0","vendor":"haxx","source":"dpkg"}]`
	return &erav1.Envelope{
		EventId:    mustULIDBytes("01J8ZK5Q8X0000000000000000"),
		ObservedAt: timestamppb.New(obs),
		IngestedAt: timestamppb.New(obs),
		Source: &erav1.Source{
			TenantId: "tenant-dev", NodeId: "node-inv-01", Hostname: "lab-host",
			AgentId: "agent-0001", AgentVersion: "0.2.0", Platform: erav1.Platform_PLATFORM_LINUX,
		},
		Payload: &erav1.Envelope_Raw{
			Raw: &erav1.RawEvent{
				SourceType: "era.inventory.host_snapshot",
				Fields: &structpb.Struct{Fields: map[string]*structpb.Value{
					"hostname":          structpb.NewStringValue("lab-host"),
					"platform":          structpb.NewStringValue("linux"),
					"os_name":           structpb.NewStringValue("Ubuntu"),
					"os_version":        structpb.NewStringValue("22.04"),
					"kernel":            structpb.NewStringValue("6.5.0"),
					"cpu_count":         structpb.NewNumberValue(8),
					"total_memory_mb":   structpb.NewNumberValue(16384),
					"software":          structpb.NewStringValue(sw),
				}},
			},
		},
	}
}

func mustULIDBytes(s string) []byte {
	var id ulid.ULID
	if err := id.UnmarshalText([]byte(s)); err != nil {
		panic(err)
	}
	return id[:]
}
