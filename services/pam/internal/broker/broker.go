// Package broker — server-side credential inject for PAM proxies (Phase 2).
// Password never leaves the server in API responses.
package broker

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Inject holds credentials bound to a session (server memory only).
type Inject struct {
	SessionID  string
	Username   string
	password   string // never exported via JSON
	SecretID   string
	CheckoutID string
	CreatedAt  time.Time
	Token      string // opaque handle for clients
}

// Store maps session → inject and token → session.
type Store struct {
	mu      sync.Mutex
	bySess  map[string]*Inject
	byToken map[string]string
}

func NewStore() *Store {
	return &Store{bySess: map[string]*Inject{}, byToken: map[string]string{}}
}

// Bind creates an inject record; password stays private.
func (s *Store) Bind(sessionID, username, password, secretID, checkoutID string) (*Inject, error) {
	if sessionID == "" || username == "" {
		return nil, fmt.Errorf("session and username required")
	}
	tok, err := randomToken()
	if err != nil {
		return nil, err
	}
	inj := &Inject{
		SessionID: sessionID, Username: username, password: password,
		SecretID: secretID, CheckoutID: checkoutID, CreatedAt: time.Now().UTC(), Token: tok,
	}
	s.mu.Lock()
	s.bySess[sessionID] = inj
	s.byToken[tok] = sessionID
	s.mu.Unlock()
	return inj, nil
}

// PublicMeta returns JSON-safe fields (no password).
func (inj *Inject) PublicMeta() map[string]any {
	if inj == nil {
		return nil
	}
	return map[string]any{
		"inject_token":   inj.Token,
		"username":       inj.Username,
		"credential_ref": hashRef(inj.SecretID),
		"injected":       true,
	}
}

// PeekUsername returns username without password.
func (s *Store) PeekUsername(sessionID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inj, ok := s.bySess[sessionID]
	if !ok {
		return "", false
	}
	return inj.Username, true
}

// ConsumePassword returns password once for gateway-side RDP handshake (lab); clears after.
func (s *Store) ConsumePassword(sessionID string) (user, pass string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inj, ok := s.bySess[sessionID]
	if !ok || inj.password == "" {
		return "", "", false
	}
	user, pass = inj.Username, inj.password
	inj.password = "" // one-shot
	return user, pass, true
}

// HasPassword reports whether inject still holds a password (tests).
func (s *Store) HasPassword(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	inj, ok := s.bySess[sessionID]
	return ok && inj.password != ""
}

func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashRef(secretID string) string {
	h := sha256.Sum256([]byte(secretID))
	return "cred:" + hex.EncodeToString(h[:8])
}
