package oidc

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
)

var ErrUserExists = errors.New("user already exists")

// identityStore abstracts user/client lookup for OIDC flows.
type identityStore interface {
	GetUserByEmail(email string) (User, bool)
	GetUserByID(id string) (User, bool)
	VerifyLogin(email, password string) bool
	ValidateRedirect(clientID, redirectURI string) bool
	// CreateUser is used by lab staging register (ERA_IDENTITY_DEV=1).
	CreateUser(tenantID, email, displayName, plainPassword string) (User, error)
}

var memUserSeq uint64

// memStore is the in-memory fallback when ERA_COMMS_DATABASE_URL is empty.
type memStore struct {
	users   map[string]User
	clients map[string][]string
}

func newMemStore() *memStore {
	s := &memStore{
		users:   make(map[string]User),
		clients: make(map[string][]string),
	}
	s.clients["era-webmail"] = []string{
		"https://app.customer.local/mail/callback",
		"http://127.0.0.1:8180/mail/callback",
		"http://localhost:8180/mail/callback",
	}
	s.users["alice@mail.gov.az"] = User{
		ID: "u-alice", TenantID: "t-demo", Email: "alice@mail.gov.az",
		Password: "1234", Roles: []string{"mail.user"},
	}
	s.users["staging@mail.gov.az"] = User{
		ID: "u-staging", TenantID: "t-demo", Email: "staging@mail.gov.az",
		Password: "staging-pass", Roles: []string{"mail.user"},
	}
	return s
}

func (m *memStore) GetUserByEmail(email string) (User, bool) {
	u, ok := m.users[email]
	return u, ok
}

func (m *memStore) GetUserByID(id string) (User, bool) {
	for _, u := range m.users {
		if u.ID == id {
			return u, true
		}
	}
	return User{}, false
}

func (m *memStore) VerifyLogin(email, password string) bool {
	u, ok := m.users[email]
	return ok && u.Password == password
}

func (m *memStore) ValidateRedirect(clientID, redirectURI string) bool {
	uris, ok := m.clients[clientID]
	if !ok {
		return false
	}
	for _, u := range uris {
		if u == redirectURI {
			return true
		}
	}
	return false
}

func (m *memStore) CreateUser(tenantID, email, displayName, plainPassword string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || plainPassword == "" {
		return User{}, fmt.Errorf("email and password required")
	}
	if _, ok := m.users[email]; ok {
		return User{}, ErrUserExists
	}
	if tenantID == "" {
		tenantID = "t-demo"
	}
	n := atomic.AddUint64(&memUserSeq, 1)
	u := User{
		ID:       fmt.Sprintf("u-reg-%d", n),
		TenantID: tenantID,
		Email:    email,
		Password: plainPassword,
		Roles:    []string{"mail.user"},
	}
	_ = displayName
	m.users[email] = u
	return u, nil
}
