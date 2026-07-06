package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func (s *postgresStore) cmdbMigrate() error {
	alters := []string{
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS fqdn TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS os_name TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS os_version TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS kernel TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS cpu_model TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS cpu_cores INTEGER DEFAULT 0`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS ram_mb BIGINT DEFAULT 0`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS disk_total_gb BIGINT DEFAULT 0`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS serial_number TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS board_serial TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS manufacturer TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS model TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS mac_addrs_json TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS ip_addrs_json TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS inventory_updated_at TIMESTAMPTZ`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS asset_kind TEXT`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS managed BOOLEAN DEFAULT TRUE`,
	}
	for _, q := range alters {
		_, _ = s.db.Exec(q)
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS asset_software (
  node_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  vendor TEXT,
  source TEXT,
  install_date TIMESTAMPTZ,
  first_seen TIMESTAMPTZ NOT NULL,
  last_seen TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (node_id, name, version)
)`,
		`CREATE TABLE IF NOT EXISTS contracts (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  vendor TEXT,
  name TEXT NOT NULL,
  start_date TIMESTAMPTZ,
  end_date TIMESTAMPTZ,
  cost_annual DOUBLE PRECISION DEFAULT 0,
  currency TEXT
)`,
		`CREATE TABLE IF NOT EXISTS software_licenses (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  product TEXT NOT NULL,
  entitled_seats INTEGER DEFAULT 0,
  contract_id TEXT
)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *postgresStore) UpsertAssetFull(a *Asset) {
	a.LastSeen = nowUTC()
	if a.InventoryUpdatedAt.IsZero() {
		a.InventoryUpdatedAt = a.LastSeen
	}
	macJSON, _ := json.Marshal(a.MACAddrs)
	ipJSON, _ := json.Marshal(a.IPAddrs)
	_, _ = s.db.Exec(`INSERT INTO assets(
		node_id,tenant_id,hostname,platform,agent_id,agent_version,last_seen,
		fqdn,os_name,os_version,kernel,cpu_model,cpu_cores,ram_mb,disk_total_gb,
		serial_number,board_serial,manufacturer,model,mac_addrs_json,ip_addrs_json,inventory_updated_at,
		asset_kind,managed)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)
		ON CONFLICT(node_id) DO UPDATE SET
		tenant_id=EXCLUDED.tenant_id, hostname=EXCLUDED.hostname, platform=EXCLUDED.platform,
		agent_id=EXCLUDED.agent_id, agent_version=EXCLUDED.agent_version, last_seen=EXCLUDED.last_seen,
		fqdn=EXCLUDED.fqdn, os_name=EXCLUDED.os_name, os_version=EXCLUDED.os_version,
		kernel=EXCLUDED.kernel, cpu_model=EXCLUDED.cpu_model, cpu_cores=EXCLUDED.cpu_cores,
		ram_mb=EXCLUDED.ram_mb, disk_total_gb=EXCLUDED.disk_total_gb,
		serial_number=EXCLUDED.serial_number, board_serial=EXCLUDED.board_serial,
		manufacturer=EXCLUDED.manufacturer, model=EXCLUDED.model,
		mac_addrs_json=EXCLUDED.mac_addrs_json, ip_addrs_json=EXCLUDED.ip_addrs_json,
		inventory_updated_at=EXCLUDED.inventory_updated_at,
		asset_kind=EXCLUDED.asset_kind, managed=EXCLUDED.managed`,
		a.NodeID, a.TenantID, a.Hostname, a.Platform, a.AgentID, a.AgentVersion, a.LastSeen,
		a.FQDN, a.OSName, a.OSVersion, a.Kernel, a.CPUModel, a.CPUCores, a.RAMMB, a.DiskTotalGB,
		a.SerialNumber, a.BoardSerial, a.Manufacturer, a.Model, string(macJSON), string(ipJSON), a.InventoryUpdatedAt,
		a.AssetKind, a.Managed)
}

func (s *postgresStore) GetAsset(nodeID string) (*Asset, bool) {
	row := s.db.QueryRow(assetSelectSQLPG+` WHERE node_id=$1`, nodeID)
	a, err := scanAssetPG(row)
	if err != nil {
		return nil, false
	}
	return a, true
}

func (s *postgresStore) FindAssetByAgentID(agentID string) (*Asset, bool) {
	if agentID == "" {
		return nil, false
	}
	row := s.db.QueryRow(assetSelectSQLPG+` WHERE agent_id=$1 LIMIT 1`, agentID)
	a, err := scanAssetPG(row)
	if err != nil {
		return nil, false
	}
	return a, true
}

func (s *postgresStore) FindAssetBySerial(serial string) (*Asset, bool) {
	if serial == "" {
		return nil, false
	}
	row := s.db.QueryRow(assetSelectSQLPG+` WHERE serial_number=$1 OR board_serial=$1 LIMIT 1`, serial)
	a, err := scanAssetPG(row)
	if err != nil {
		return nil, false
	}
	return a, true
}

const assetSelectSQLPG = `SELECT node_id, tenant_id, hostname, platform, agent_id, agent_version, last_seen,
		fqdn, os_name, os_version, kernel, cpu_model, cpu_cores, ram_mb, disk_total_gb,
		serial_number, board_serial, manufacturer, model, mac_addrs_json, ip_addrs_json, inventory_updated_at,
		asset_kind, managed
		FROM assets`

func scanAssetPG(row *sql.Row) (*Asset, error) {
	var a Asset
	var macJ, ipJ sql.NullString
	var cpuC, ram, disk sql.NullInt64
	var invTS sql.NullTime
	var assetKind sql.NullString
	var managed sql.NullBool
	if err := row.Scan(&a.NodeID, &a.TenantID, &a.Hostname, &a.Platform, &a.AgentID, &a.AgentVersion, &a.LastSeen,
		&a.FQDN, &a.OSName, &a.OSVersion, &a.Kernel, &a.CPUModel, &cpuC, &ram, &disk,
		&a.SerialNumber, &a.BoardSerial, &a.Manufacturer, &a.Model, &macJ, &ipJ, &invTS,
		&assetKind, &managed); err != nil {
		return nil, err
	}
	if invTS.Valid {
		a.InventoryUpdatedAt = invTS.Time
	}
	if cpuC.Valid {
		a.CPUCores = uint32(cpuC.Int64)
	}
	if ram.Valid {
		a.RAMMB = uint64(ram.Int64)
	}
	if disk.Valid {
		a.DiskTotalGB = uint64(disk.Int64)
	}
	if macJ.Valid {
		_ = json.Unmarshal([]byte(macJ.String), &a.MACAddrs)
	}
	if ipJ.Valid {
		_ = json.Unmarshal([]byte(ipJ.String), &a.IPAddrs)
	}
	if assetKind.Valid {
		a.AssetKind = assetKind.String
	}
	if managed.Valid {
		a.Managed = managed.Bool
	} else if a.AgentID != "" {
		a.Managed = true
	}
	return &a, nil
}

func (s *postgresStore) ReplaceAssetSoftware(nodeID, tenantID string, items []*AssetSoftware) {
	_, _ = s.db.Exec(`DELETE FROM asset_software WHERE node_id=$1`, nodeID)
	now := nowUTC()
	for _, it := range items {
		if it == nil {
			continue
		}
		fs := it.FirstSeen
		if fs.IsZero() {
			fs = now
		}
		_, _ = s.db.Exec(`INSERT INTO asset_software(node_id,tenant_id,name,version,vendor,source,install_date,first_seen,last_seen)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			nodeID, tenantID, it.Name, it.Version, it.Vendor, it.Source, nullTime(it.InstallDate), fs, now)
	}
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func (s *postgresStore) ListAssetSoftware(nodeID string) []*AssetSoftware {
	rows, err := s.db.Query(`SELECT node_id,tenant_id,name,version,vendor,source,install_date,first_seen,last_seen
		FROM asset_software WHERE node_id=$1`, nodeID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanSoftwarePG(rows)
}

func (s *postgresStore) ListAllAssetSoftware() []*AssetSoftware {
	q := `SELECT node_id,tenant_id,name,version,vendor,source,install_date,first_seen,last_seen FROM asset_software`
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = $1`
		args = append(args, s.tenantFilter)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanSoftwarePG(rows)
}

func scanSoftwarePG(rows *sql.Rows) []*AssetSoftware {
	var out []*AssetSoftware
	for rows.Next() {
		var sw AssetSoftware
		var inst sql.NullTime
		if rows.Scan(&sw.NodeID, &sw.TenantID, &sw.Name, &sw.Version, &sw.Vendor, &sw.Source, &inst, &sw.FirstSeen, &sw.LastSeen) != nil {
			continue
		}
		if inst.Valid {
			sw.InstallDate = inst.Time
		}
		out = append(out, &sw)
	}
	return out
}

func (s *postgresStore) UpsertContract(c *Contract) {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	_, _ = s.db.Exec(`INSERT INTO contracts(id,tenant_id,vendor,name,start_date,end_date,cost_annual,currency)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO UPDATE SET
		tenant_id=EXCLUDED.tenant_id, vendor=EXCLUDED.vendor, name=EXCLUDED.name,
		start_date=EXCLUDED.start_date, end_date=EXCLUDED.end_date,
		cost_annual=EXCLUDED.cost_annual, currency=EXCLUDED.currency`,
		c.ID, c.TenantID, c.Vendor, c.Name, nullTime(c.StartDate), nullTime(c.EndDate), c.CostAnnual, c.Currency)
}

func (s *postgresStore) ListContracts() []*Contract {
	q := `SELECT id,tenant_id,vendor,name,start_date,end_date,cost_annual,currency FROM contracts`
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = $1`
		args = append(args, s.tenantFilter)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Contract
	for rows.Next() {
		var c Contract
		var sd, ed sql.NullTime
		if rows.Scan(&c.ID, &c.TenantID, &c.Vendor, &c.Name, &sd, &ed, &c.CostAnnual, &c.Currency) != nil {
			continue
		}
		if sd.Valid {
			c.StartDate = sd.Time
		}
		if ed.Valid {
			c.EndDate = ed.Time
		}
		out = append(out, &c)
	}
	return out
}

func (s *postgresStore) UpsertSoftwareLicense(l *SoftwareLicense) {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	_, _ = s.db.Exec(`INSERT INTO software_licenses(id,tenant_id,product,entitled_seats,contract_id)
		VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET
		tenant_id=EXCLUDED.tenant_id, product=EXCLUDED.product,
		entitled_seats=EXCLUDED.entitled_seats, contract_id=EXCLUDED.contract_id`,
		l.ID, l.TenantID, l.Product, l.EntitledSeats, l.ContractID)
}

func (s *postgresStore) ListSoftwareLicenses() []*SoftwareLicense {
	q := `SELECT id,tenant_id,product,entitled_seats,contract_id FROM software_licenses`
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = $1`
		args = append(args, s.tenantFilter)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*SoftwareLicense
	for rows.Next() {
		var l SoftwareLicense
		if rows.Scan(&l.ID, &l.TenantID, &l.Product, &l.EntitledSeats, &l.ContractID) != nil {
			continue
		}
		out = append(out, &l)
	}
	return out
}

func (s *postgresStore) ReconcileSoftwareLicenses() []ReconcileRow {
	return reconcileInstalledEntitled(s.ListAllAssetSoftware(), s.ListSoftwareLicenses())
}
