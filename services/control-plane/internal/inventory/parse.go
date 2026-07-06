package inventory

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// SnapshotFromEnvelope извлекает Snapshot из Envelope RawEvent inventory.
func SnapshotFromEnvelope(env *erav1.Envelope) (Snapshot, bool) {
	if env == nil {
		return Snapshot{}, false
	}
	raw := env.GetRaw()
	if raw == nil || !strings.HasPrefix(raw.GetSourceType(), "era.inventory") {
		return Snapshot{}, false
	}
	src := env.GetSource()
	snap := Snapshot{
		TenantID:     src.GetTenantId(),
		NodeID:       src.GetNodeId(),
		AgentID:      src.GetAgentId(),
		AgentVersion: src.GetAgentVersion(),
		ObservedAt:   env.GetObservedAt().AsTime().UTC(),
	}
	fields := raw.GetFields().GetFields()
	snap.Hostname = strField(fields, "hostname")
	snap.Platform = strField(fields, "platform")
	snap.FQDN = strField(fields, "fqdn")
	snap.OSName = strField(fields, "os_name")
	snap.OSVersion = strField(fields, "os_version")
	snap.Kernel = strField(fields, "kernel")
	snap.CPUModel = strField(fields, "cpu_model")
	snap.CPUCores = uint32(numField(fields, "cpu_count"))
	snap.RAMMB = uint64(numField(fields, "total_memory_mb"))
	snap.DiskTotalGB = uint64(numField(fields, "disk_total_gb"))
	snap.SerialNumber = strField(fields, "serial_number")
	snap.BoardSerial = strField(fields, "board_serial")
	snap.Manufacturer = strField(fields, "manufacturer")
	snap.Model = strField(fields, "model")
	snap.MACAddrs = jsonStringSlice(strField(fields, "mac_addrs"))
	snap.IPAddrs = jsonStringSlice(strField(fields, "ip_addrs"))
	if sw := strField(fields, "software"); sw != "" {
		_ = json.Unmarshal([]byte(sw), &snap.Software)
	}
	if snap.Hostname == "" {
		snap.Hostname = src.GetHostname()
	}
	return snap, true
}

func strField(fields map[string]*structpb.Value, key string) string {
	if fields == nil {
		return ""
	}
	v, ok := fields[key]
	if !ok || v == nil {
		return ""
	}
	return v.GetStringValue()
}

func numField(fields map[string]*structpb.Value, key string) float64 {
	if fields == nil {
		return 0
	}
	v, ok := fields[key]
	if !ok || v == nil {
		return 0
	}
	return v.GetNumberValue()
}

func jsonStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// ParseSoftwareJSON — helper for tests.
func ParseSoftwareJSON(s string) []SoftwareItem {
	var out []SoftwareItem
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

// FormatObserved — test helper.
func FormatObserved(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}
