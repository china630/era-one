// Package inventory — Kafka consumer + CMDB merge (ADR-0011).
package inventory

import (
	"strings"
	"time"

	"era/services/control-plane/internal/store"
)

// Snapshot — нормализованный inventory снимок с агента.
type Snapshot struct {
	NodeID        string
	TenantID      string
	AgentID       string
	AgentVersion  string
	Hostname      string
	Platform      string
	FQDN          string
	OSName        string
	OSVersion     string
	Kernel        string
	CPUModel      string
	CPUCores      uint32
	RAMMB         uint64
	DiskTotalGB   uint64
	SerialNumber  string
	BoardSerial   string
	Manufacturer  string
	Model         string
	MACAddrs      []string
	IPAddrs       []string
	Software      []SoftwareItem
	ObservedAt    time.Time
}

type SoftwareItem struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Vendor  string `json:"vendor"`
	Source  string `json:"source"`
}

// ResolveNodeID — merge/dedup: agent_id > serial > MAC > hostname+tenant.
func ResolveNodeID(st store.Repository, snap Snapshot) (nodeID, rule string) {
	if snap.AgentID != "" {
		if a, ok := st.FindAssetByAgentID(snap.AgentID); ok {
			return a.NodeID, "agent_id"
		}
	}
	serial := snap.SerialNumber
	if serial == "" {
		serial = snap.BoardSerial
	}
	if serial != "" {
		if a, ok := st.FindAssetBySerial(serial); ok {
			return a.NodeID, "serial"
		}
	}
	if len(snap.MACAddrs) > 0 {
		for _, a := range st.ListAssets() {
			if macOverlap(a.MACAddrs, snap.MACAddrs) {
				return a.NodeID, "mac"
			}
		}
	}
	if snap.Hostname != "" && snap.TenantID != "" {
		for _, a := range st.ListAssets() {
			if a.Hostname == snap.Hostname && a.TenantID == snap.TenantID {
				return a.NodeID, "hostname"
			}
		}
	}
	if snap.NodeID != "" {
		return snap.NodeID, "new"
	}
	return snap.NodeID, "new"
}

// ApplySnapshot upsert CMDB + software; возвращает audit при конфликте hostname/node.
func ApplySnapshot(st store.Repository, snap Snapshot) (nodeID, rule, audit string) {
	nodeID, rule = ResolveNodeID(st, snap)
	if nodeID == "" {
		nodeID = snap.NodeID
	}
	if nodeID == "" {
		return "", rule, "missing node_id"
	}
	if existing, ok := st.GetAsset(nodeID); ok {
		if snap.Hostname != "" && existing.Hostname != "" && existing.Hostname != snap.Hostname {
			audit = "cmdb.merge_conflict hostname " + existing.Hostname + " -> " + snap.Hostname
		}
	}
	now := snap.ObservedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	a := &store.Asset{
		NodeID: nodeID, TenantID: snap.TenantID, Hostname: snap.Hostname,
		Platform: snap.Platform, AgentID: snap.AgentID, AgentVersion: snap.AgentVersion,
		LastSeen: now, FQDN: snap.FQDN, OSName: snap.OSName, OSVersion: snap.OSVersion,
		Kernel: snap.Kernel, CPUModel: snap.CPUModel, CPUCores: snap.CPUCores,
		RAMMB: snap.RAMMB, DiskTotalGB: snap.DiskTotalGB,
		SerialNumber: snap.SerialNumber, BoardSerial: snap.BoardSerial,
		Manufacturer: snap.Manufacturer, Model: snap.Model,
		MACAddrs: snap.MACAddrs, IPAddrs: snap.IPAddrs,
		InventoryUpdatedAt: now,
	}
	st.UpsertAssetFull(a)
	var sw []*store.AssetSoftware
	for _, item := range snap.Software {
		sw = append(sw, &store.AssetSoftware{
			Name: item.Name, Version: item.Version, Vendor: item.Vendor, Source: item.Source,
		})
	}
	st.ReplaceAssetSoftware(nodeID, snap.TenantID, sw)
	return nodeID, rule, audit
}

func macOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, m := range a {
		set[strings.ToLower(m)] = struct{}{}
	}
	for _, m := range b {
		if _, ok := set[strings.ToLower(m)]; ok {
			return true
		}
	}
	return false
}
