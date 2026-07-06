package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func (s *sqliteStore) cmdbMigrate() error {
	alters := []string{
		`ALTER TABLE assets ADD COLUMN fqdn TEXT`,
		`ALTER TABLE assets ADD COLUMN os_name TEXT`,
		`ALTER TABLE assets ADD COLUMN os_version TEXT`,
		`ALTER TABLE assets ADD COLUMN kernel TEXT`,
		`ALTER TABLE assets ADD COLUMN cpu_model TEXT`,
		`ALTER TABLE assets ADD COLUMN cpu_cores INTEGER DEFAULT 0`,
		`ALTER TABLE assets ADD COLUMN ram_mb INTEGER DEFAULT 0`,
		`ALTER TABLE assets ADD COLUMN disk_total_gb INTEGER DEFAULT 0`,
		`ALTER TABLE assets ADD COLUMN serial_number TEXT`,
		`ALTER TABLE assets ADD COLUMN board_serial TEXT`,
		`ALTER TABLE assets ADD COLUMN manufacturer TEXT`,
		`ALTER TABLE assets ADD COLUMN model TEXT`,
		`ALTER TABLE assets ADD COLUMN mac_addrs_json TEXT`,
		`ALTER TABLE assets ADD COLUMN ip_addrs_json TEXT`,
		`ALTER TABLE assets ADD COLUMN inventory_updated_at TEXT`,
		`ALTER TABLE assets ADD COLUMN asset_kind TEXT`,
		`ALTER TABLE assets ADD COLUMN managed INTEGER DEFAULT 1`,
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
  install_date TEXT,
  first_seen TEXT NOT NULL,
  last_seen TEXT NOT NULL,
  PRIMARY KEY (node_id, name, version)
)`,
		`CREATE TABLE IF NOT EXISTS contracts (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  vendor TEXT,
  name TEXT NOT NULL,
  start_date TEXT,
  end_date TEXT,
  cost_annual REAL DEFAULT 0,
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

func (s *sqliteStore) UpsertAssetFull(a *Asset) {
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
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(node_id) DO UPDATE SET
		tenant_id=excluded.tenant_id, hostname=excluded.hostname, platform=excluded.platform,
		agent_id=excluded.agent_id, agent_version=excluded.agent_version, last_seen=excluded.last_seen,
		fqdn=excluded.fqdn, os_name=excluded.os_name, os_version=excluded.os_version,
		kernel=excluded.kernel, cpu_model=excluded.cpu_model, cpu_cores=excluded.cpu_cores,
		ram_mb=excluded.ram_mb, disk_total_gb=excluded.disk_total_gb,
		serial_number=excluded.serial_number, board_serial=excluded.board_serial,
		manufacturer=excluded.manufacturer, model=excluded.model,
		mac_addrs_json=excluded.mac_addrs_json, ip_addrs_json=excluded.ip_addrs_json,
		inventory_updated_at=excluded.inventory_updated_at,
		asset_kind=excluded.asset_kind, managed=excluded.managed`,
		a.NodeID, a.TenantID, a.Hostname, a.Platform, a.AgentID, a.AgentVersion, a.LastSeen.Format(time.RFC3339Nano),
		a.FQDN, a.OSName, a.OSVersion, a.Kernel, a.CPUModel, a.CPUCores, a.RAMMB, a.DiskTotalGB,
		a.SerialNumber, a.BoardSerial, a.Manufacturer, a.Model, string(macJSON), string(ipJSON),
		a.InventoryUpdatedAt.Format(time.RFC3339Nano), a.AssetKind, boolToInt(a.Managed))
}

func (s *sqliteStore) GetAsset(nodeID string) (*Asset, bool) {
	row := s.db.QueryRow(assetSelectSQL+` WHERE node_id=?`, nodeID)
	a, err := scanAssetSQLite(row)
	if err != nil {
		return nil, false
	}
	return a, true
}

func (s *sqliteStore) FindAssetByAgentID(agentID string) (*Asset, bool) {
	if agentID == "" {
		return nil, false
	}
	row := s.db.QueryRow(assetSelectSQL+` WHERE agent_id=? LIMIT 1`, agentID)
	a, err := scanAssetSQLite(row)
	if err != nil {
		return nil, false
	}
	return a, true
}

func (s *sqliteStore) FindAssetBySerial(serial string) (*Asset, bool) {
	if serial == "" {
		return nil, false
	}
	row := s.db.QueryRow(assetSelectSQL+` WHERE serial_number=? OR board_serial=? LIMIT 1`, serial, serial)
	a, err := scanAssetSQLite(row)
	if err != nil {
		return nil, false
	}
	return a, true
}

const assetSelectSQL = `SELECT node_id, tenant_id, hostname, platform, agent_id, agent_version, last_seen,
		fqdn, os_name, os_version, kernel, cpu_model, cpu_cores, ram_mb, disk_total_gb,
		serial_number, board_serial, manufacturer, model, mac_addrs_json, ip_addrs_json, inventory_updated_at,
		asset_kind, managed
		FROM assets`

func scanAssetSQLite(row *sql.Row) (*Asset, error) {
	var a Asset
	var ts, invTS, macJ, ipJ sql.NullString
	var cpuC, ram, disk, managed sql.NullInt64
	var assetKind sql.NullString
	if err := row.Scan(&a.NodeID, &a.TenantID, &a.Hostname, &a.Platform, &a.AgentID, &a.AgentVersion, &ts,
		&a.FQDN, &a.OSName, &a.OSVersion, &a.Kernel, &a.CPUModel, &cpuC, &ram, &disk,
		&a.SerialNumber, &a.BoardSerial, &a.Manufacturer, &a.Model, &macJ, &ipJ, &invTS,
		&assetKind, &managed); err != nil {
		return nil, err
	}
	a.LastSeen, _ = time.Parse(time.RFC3339Nano, ts.String)
	if invTS.Valid {
		a.InventoryUpdatedAt, _ = time.Parse(time.RFC3339Nano, invTS.String)
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
		a.Managed = managed.Int64 != 0
	} else if a.AgentID != "" {
		a.Managed = true
	}
	return &a, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *sqliteStore) ReplaceAssetSoftware(nodeID, tenantID string, items []*AssetSoftware) {
	_, _ = s.db.Exec(`DELETE FROM asset_software WHERE node_id=?`, nodeID)
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
			VALUES(?,?,?,?,?,?,?,?,?)`,
			nodeID, tenantID, it.Name, it.Version, it.Vendor, it.Source,
			formatOptTime(it.InstallDate), fs.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	}
}

func (s *sqliteStore) ListAssetSoftware(nodeID string) []*AssetSoftware {
	rows, err := s.db.Query(`SELECT node_id,tenant_id,name,version,vendor,source,install_date,first_seen,last_seen
		FROM asset_software WHERE node_id=?`, nodeID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanSoftwareSQLite(rows)
}

func (s *sqliteStore) ListAllAssetSoftware() []*AssetSoftware {
	q := `SELECT node_id,tenant_id,name,version,vendor,source,install_date,first_seen,last_seen FROM asset_software`
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = ?`
		args = append(args, s.tenantFilter)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanSoftwareSQLite(rows)
}

func scanSoftwareSQLite(rows *sql.Rows) []*AssetSoftware {
	var out []*AssetSoftware
	for rows.Next() {
		var sw AssetSoftware
		var inst, fs, ls string
		if rows.Scan(&sw.NodeID, &sw.TenantID, &sw.Name, &sw.Version, &sw.Vendor, &sw.Source, &inst, &fs, &ls) != nil {
			continue
		}
		sw.InstallDate, _ = time.Parse(time.RFC3339Nano, inst)
		sw.FirstSeen, _ = time.Parse(time.RFC3339Nano, fs)
		sw.LastSeen, _ = time.Parse(time.RFC3339Nano, ls)
		out = append(out, &sw)
	}
	return out
}

func (s *sqliteStore) UpsertContract(c *Contract) {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	_, _ = s.db.Exec(`INSERT INTO contracts(id,tenant_id,vendor,name,start_date,end_date,cost_annual,currency)
		VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET
		tenant_id=excluded.tenant_id, vendor=excluded.vendor, name=excluded.name,
		start_date=excluded.start_date, end_date=excluded.end_date,
		cost_annual=excluded.cost_annual, currency=excluded.currency`,
		c.ID, c.TenantID, c.Vendor, c.Name,
		formatOptTime(c.StartDate), formatOptTime(c.EndDate), c.CostAnnual, c.Currency)
}

func (s *sqliteStore) ListContracts() []*Contract {
	q := `SELECT id,tenant_id,vendor,name,start_date,end_date,cost_annual,currency FROM contracts`
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = ?`
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
		var sd, ed sql.NullString
		if rows.Scan(&c.ID, &c.TenantID, &c.Vendor, &c.Name, &sd, &ed, &c.CostAnnual, &c.Currency) != nil {
			continue
		}
		if sd.Valid {
			c.StartDate, _ = time.Parse(time.RFC3339Nano, sd.String)
		}
		if ed.Valid {
			c.EndDate, _ = time.Parse(time.RFC3339Nano, ed.String)
		}
		out = append(out, &c)
	}
	return out
}

func (s *sqliteStore) UpsertSoftwareLicense(l *SoftwareLicense) {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	_, _ = s.db.Exec(`INSERT INTO software_licenses(id,tenant_id,product,entitled_seats,contract_id)
		VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET
		tenant_id=excluded.tenant_id, product=excluded.product,
		entitled_seats=excluded.entitled_seats, contract_id=excluded.contract_id`,
		l.ID, l.TenantID, l.Product, l.EntitledSeats, l.ContractID)
}

func (s *sqliteStore) ListSoftwareLicenses() []*SoftwareLicense {
	q := `SELECT id,tenant_id,product,entitled_seats,contract_id FROM software_licenses`
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = ?`
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

func (s *sqliteStore) ReconcileSoftwareLicenses() []ReconcileRow {
	return reconcileInstalledEntitled(s.ListAllAssetSoftware(), s.ListSoftwareLicenses())
}

func formatOptTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
