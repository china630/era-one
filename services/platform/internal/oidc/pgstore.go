package oidc

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PGStore persists identity users and OIDC clients in era_comms schema.
type PGStore struct {
	db *sql.DB
}

// OpenPGStore connects using ERA_COMMS_DATABASE_URL DSN.
func OpenPGStore(dsn string) (*PGStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &PGStore{db: db}, nil
}

// Close releases the database connection.
func (s *PGStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// SeedDefaults inserts demo OIDC client and user when missing.
func (s *PGStore) SeedDefaults() error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO era_comms.tenants (id, name, slug) VALUES ('t-demo','Demo Tenant','t-demo') ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("seed tenant: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO era_comms.oidc_clients (client_id, client_secret, redirect_uris)
		 VALUES ('era-webmail','dev-secret', ARRAY['https://app.customer.local/mail/callback','http://127.0.0.1:8180/mail/callback','http://localhost:8180/mail/callback'])
		 ON CONFLICT (client_id) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("seed client: %w", err)
	}
	hash, err := hashPassword("1234")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO era_comms.identity_users (id, tenant_id, email, display_name, password_hash, active)
		 VALUES ('u-alice','t-demo','alice@mail.gov.az','Alice Demo',$1,true)
		 ON CONFLICT (email) DO NOTHING`, hash)
	if err != nil {
		return fmt.Errorf("seed user: %w", err)
	}
	stagingHash, err := hashPassword("staging-pass")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO era_comms.identity_users (id, tenant_id, email, display_name, password_hash, active)
		 VALUES ('u-staging','t-demo','staging@mail.gov.az','Staging Pilot',$1,true)
		 ON CONFLICT (email) DO NOTHING`, stagingHash)
	if err != nil {
		return fmt.Errorf("seed staging user: %w", err)
	}
	return nil
}

// CreateUser inserts a new identity user.
func (s *PGStore) CreateUser(tenantID, email, displayName, plainPassword string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	hash, err := hashPassword(plainPassword)
	if err != nil {
		return User{}, err
	}
	id := uuid.NewString()
	_, err = s.db.Exec(
		`INSERT INTO era_comms.identity_users (id, tenant_id, email, display_name, password_hash, active)
		 VALUES ($1,$2,$3,$4,$5,true)`,
		id, tenantID, email, displayName, hash,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
			strings.Contains(err.Error(), "unique") {
			return User{}, ErrUserExists
		}
		return User{}, err
	}
	return User{ID: id, TenantID: tenantID, Email: email, Roles: []string{"mail.user"}}, nil
}

// GetUserByEmail returns an active user by email.
func (s *PGStore) GetUserByEmail(email string) (User, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	var u User
	var display string
	err := s.db.QueryRow(
		`SELECT id, tenant_id, email, display_name FROM era_comms.identity_users WHERE email=$1 AND active`,
		email,
	).Scan(&u.ID, &u.TenantID, &u.Email, &display)
	if err != nil {
		return User{}, false
	}
	u.Roles = []string{"mail.user"}
	return u, true
}

// GetUserByID returns an active user by id.
func (s *PGStore) GetUserByID(id string) (User, bool) {
	var u User
	var display string
	err := s.db.QueryRow(
		`SELECT id, tenant_id, email, display_name FROM era_comms.identity_users WHERE id=$1 AND active`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &display)
	if err != nil {
		return User{}, false
	}
	u.Roles = []string{"mail.user"}
	return u, true
}

// ListUsersByTenant returns users for a tenant.
func (s *PGStore) ListUsersByTenant(tenantID string) ([]User, error) {
	rows, err := s.db.Query(
		`SELECT id, tenant_id, email FROM era_comms.identity_users WHERE tenant_id=$1 AND active ORDER BY email`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email); err != nil {
			return nil, err
		}
		u.Roles = []string{"mail.user"}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UpdateUser updates display name and/or password for a user.
func (s *PGStore) UpdateUser(email, displayName, plainPassword string, active *bool) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if plainPassword != "" {
		hash, err := hashPassword(plainPassword)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(
			`UPDATE era_comms.identity_users SET display_name=COALESCE(NULLIF($2,''), display_name), password_hash=$3 WHERE email=$1`,
			email, displayName, hash,
		)
		return err
	}
	if active != nil {
		_, err := s.db.Exec(
			`UPDATE era_comms.identity_users SET display_name=COALESCE(NULLIF($2,''), display_name), active=$3 WHERE email=$1`,
			email, displayName, *active,
		)
		return err
	}
	_, err := s.db.Exec(
		`UPDATE era_comms.identity_users SET display_name=COALESCE(NULLIF($2,''), display_name) WHERE email=$1`,
		email, displayName,
	)
	return err
}

// DeleteUser deactivates a user (soft delete).
func (s *PGStore) DeleteUser(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	_, err := s.db.Exec(`UPDATE era_comms.identity_users SET active=false WHERE email=$1`, email)
	return err
}

// VerifyLogin checks email/password against stored hash.
func (s *PGStore) VerifyLogin(email, password string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	var hash string
	err := s.db.QueryRow(
		`SELECT password_hash FROM era_comms.identity_users WHERE email=$1 AND active`, email,
	).Scan(&hash)
	if err != nil {
		return false
	}
	return verifyPassword(hash, password)
}

// ValidateRedirect returns true when redirect URI is registered for the client.
func (s *PGStore) ValidateRedirect(clientID, redirectURI string) bool {
	var ok int
	err := s.db.QueryRow(
		`SELECT 1 FROM era_comms.oidc_clients WHERE client_id=$1 AND $2 = ANY(redirect_uris)`,
		clientID, redirectURI,
	).Scan(&ok)
	return err == nil
}

// CreateClient registers an OIDC client.
func (s *PGStore) CreateClient(clientID, secret string, redirectURIs []string) error {
	_, err := s.db.Exec(
		`INSERT INTO era_comms.oidc_clients (client_id, client_secret, redirect_uris) VALUES ($1,$2,$3::text[])`,
		clientID, secret, pgTextArray(redirectURIs),
	)
	return err
}

// GetClient returns client metadata.
func (s *PGStore) GetClient(clientID string) (id, secret string, redirectURIs []string, ok bool) {
	var uris string
	err := s.db.QueryRow(
		`SELECT client_id, client_secret, array_to_string(redirect_uris, '\x1e') FROM era_comms.oidc_clients WHERE client_id=$1`, clientID,
	).Scan(&id, &secret, &uris)
	if err != nil {
		return "", "", nil, false
	}
	if uris != "" {
		redirectURIs = strings.Split(uris, "\x1e")
	}
	return id, secret, redirectURIs, true
}

// UpdateClient replaces redirect URIs and optional secret.
func (s *PGStore) UpdateClient(clientID, secret string, redirectURIs []string) error {
	if secret != "" {
		_, err := s.db.Exec(
			`UPDATE era_comms.oidc_clients SET client_secret=$2, redirect_uris=$3::text[] WHERE client_id=$1`,
			clientID, secret, pgTextArray(redirectURIs),
		)
		return err
	}
	_, err := s.db.Exec(
		`UPDATE era_comms.oidc_clients SET redirect_uris=$2::text[] WHERE client_id=$1`,
		clientID, pgTextArray(redirectURIs),
	)
	return err
}

// DeleteClient removes an OIDC client.
func (s *PGStore) DeleteClient(clientID string) error {
	_, err := s.db.Exec(`DELETE FROM era_comms.oidc_clients WHERE client_id=$1`, clientID)
	return err
}

// ListClients returns all registered OIDC clients (without secrets).
func (s *PGStore) ListClients() ([]string, error) {
	rows, err := s.db.Query(`SELECT client_id FROM era_comms.oidc_clients ORDER BY client_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func pgTextArray(elems []string) string {
	if len(elems) == 0 {
		return "{}"
	}
	quoted := make([]string, len(elems))
	for i, e := range elems {
		quoted[i] = `"` + strings.ReplaceAll(e, `"`, `\"`) + `"`
	}
	return "{" + strings.Join(quoted, ",") + "}"
}
