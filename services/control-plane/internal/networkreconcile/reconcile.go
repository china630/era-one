// Package networkreconcile — CMDB reconciliation для сетевых устройств без агента (ADR-0020).
package networkreconcile

import (
	"strings"

	"era/services/control-plane/internal/store"
)

// Input — сетевой актив от ERA Observe.
type Input struct {
	NodeID   string
	TenantID string
	Hostname string
	Kind     string
	IPAddrs  []string
	MACAddrs []string
}

// Result — итог upsert.
type Result struct {
	Asset    *store.Asset
	Conflict bool
	Audit    string
}

// Upsert регистрирует unmanaged network asset с дедупом MAC/IP/hostname.
func Upsert(st store.Repository, in Input) Result {
	if in.NodeID == "" || in.TenantID == "" {
		return Result{Audit: "missing node_id or tenant_id"}
	}
	if conflict, audit := managedIPConflict(st, in); conflict {
		return Result{Conflict: true, Audit: audit}
	}
	if existing := findNetwork(st, in); existing != nil {
		mergeNetwork(existing, in)
		st.UpsertAssetFull(existing)
		return Result{Asset: existing}
	}
	a := &store.Asset{
		NodeID: in.NodeID, TenantID: in.TenantID,
		Hostname: firstNonEmpty(in.Hostname, in.NodeID),
		Platform: "network", Model: in.Kind,
		AssetKind: firstNonEmpty(in.Kind, "unknown"),
		Managed:   false,
		IPAddrs: append([]string(nil), in.IPAddrs...),
		MACAddrs: append([]string(nil), in.MACAddrs...),
	}
	st.UpsertAssetFull(a)
	return Result{Asset: a}
}

func ListNetwork(st store.Repository) []*store.Asset {
	var out []*store.Asset
	for _, a := range st.ListAssets() {
		if a.Platform == "network" && a.AgentID == "" {
			out = append(out, a)
		}
	}
	return out
}

func managedIPConflict(st store.Repository, in Input) (bool, string) {
	for _, a := range st.ListAssets() {
		if a.AgentID == "" {
			continue
		}
		for _, ip := range in.IPAddrs {
			if ip != "" && hasIP(a, ip) {
				return true, "cmdb.observe_conflict managed endpoint " + a.NodeID + " owns " + ip
			}
		}
	}
	return false, ""
}

func findNetwork(st store.Repository, in Input) *store.Asset {
	for _, a := range st.ListAssets() {
		if a.Platform != "network" || a.AgentID != "" {
			continue
		}
		if a.NodeID == in.NodeID {
			return a
		}
		if in.Hostname != "" && strings.EqualFold(a.Hostname, in.Hostname) {
			return a
		}
		for _, mac := range in.MACAddrs {
			if mac != "" && hasMAC(a, mac) {
				return a
			}
		}
		for _, ip := range in.IPAddrs {
			if ip != "" && hasIP(a, ip) {
				return a
			}
		}
	}
	return nil
}

func mergeNetwork(a *store.Asset, in Input) {
	if in.Hostname != "" {
		a.Hostname = in.Hostname
	}
	if in.Kind != "" {
		a.Model = in.Kind
		a.AssetKind = in.Kind
	}
	a.Managed = false
	a.IPAddrs = unionStrings(a.IPAddrs, in.IPAddrs)
	a.MACAddrs = unionStrings(a.MACAddrs, in.MACAddrs)
}

func hasIP(a *store.Asset, ip string) bool {
	for _, x := range a.IPAddrs {
		if x == ip {
			return true
		}
	}
	return false
}

func hasMAC(a *store.Asset, mac string) bool {
	mac = strings.ToLower(mac)
	for _, x := range a.MACAddrs {
		if strings.ToLower(x) == mac {
			return true
		}
	}
	return false
}

func unionStrings(a, b []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range append(append([]string{}, a...), b...) {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
