// Package oidc — minimal OIDC provider for ERA webmail pilot (R-3).

package oidc



import (

	"crypto/rand"

	"crypto/sha256"

	"encoding/base64"

	"encoding/json"

	"net/http"

	"net/url"

	"strings"

	"sync"

	"time"

	"github.com/google/uuid"

)



// User is an identity account.

type User struct {

	ID       string

	TenantID string

	Email    string

	Password string // in-memory demo only; empty when loaded from Postgres

	Roles    []string

}



// Server implements authorization code + PKCE subset.

type Server struct {

	mu     sync.RWMutex

	store  identityStore

	codes  map[string]authCode

	secret []byte

	issuer string

}



type authCode struct {

	ClientID    string

	RedirectURI string

	Challenge   string

	Method      string

	UserID      string

	Expires     time.Time

}



// NewServer creates OIDC provider. Uses Postgres when dbURL is set, else in-memory demo store.

func NewServer(issuer string, secret []byte, dbURL string) (*Server, error) {

	s := &Server{

		codes:  make(map[string]authCode),

		secret: secret,

		issuer: issuer,

	}

	if strings.TrimSpace(dbURL) == "" {

		s.store = newMemStore()

		return s, nil

	}

	pg, err := OpenPGStore(dbURL)

	if err != nil {

		return nil, err

	}

	if err := pg.SeedDefaults(); err != nil {

		_ = pg.Close()

		return nil, err

	}

	s.store = pg

	return s, nil

}



// Register mounts OIDC routes.

func (s *Server) Register(mux *http.ServeMux) {

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {

		w.Write([]byte(`{"status":"ok"}`))

	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {

		w.Write([]byte(`{"ready":true}`))

	})

	mux.HandleFunc("/.well-known/openid-configuration", s.discovery)

	mux.HandleFunc("/oauth2/authorize", s.authorize)

	mux.HandleFunc("/oauth2/token", s.token)

	mux.HandleFunc("/oauth2/login", s.loginForm)

	mux.HandleFunc("/oauth2/login/submit", s.loginSubmit)

	s.registerStaging(mux)

}



func (s *Server) discovery(w http.ResponseWriter, _ *http.Request) {

	writeJSON(w, map[string]any{

		"issuer":                           s.issuer,

		"authorization_endpoint":           s.issuer + "/oauth2/authorize",

		"token_endpoint":                   s.issuer + "/oauth2/token",

		"response_types_supported":         []string{"code"},

		"code_challenge_methods_supported": []string{"S256"},

		"grant_types_supported":            []string{"authorization_code"},

	})

}



func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {

	q := r.URL.Query()

	clientID := q.Get("client_id")

	redirectURI := q.Get("redirect_uri")

	challenge := q.Get("code_challenge")

	method := q.Get("code_challenge_method")

	state := q.Get("state")

	if clientID == "" || redirectURI == "" || challenge == "" {

		http.Error(w, "invalid request", http.StatusBadRequest)

		return

	}

	if !s.store.ValidateRedirect(clientID, redirectURI) {

		http.Error(w, "invalid redirect", http.StatusBadRequest)

		return

	}

	http.Redirect(w, r, "/oauth2/login?"+url.Values{

		"client_id":             {clientID},

		"redirect_uri":          {redirectURI},

		"code_challenge":        {challenge},

		"code_challenge_method": {method},

		"state":                 {state},

	}.Encode(), http.StatusFound)

}



func (s *Server) loginForm(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>

<h1>ERA Mail Login</h1>

<form method="POST" action="/oauth2/login/submit?` + r.URL.RawQuery + `">

<label>Email <input name="email" type="email"/></label>

<label>Password <input name="password" type="password"/></label>

<button type="submit">Sign in</button>

</form></body></html>`))

}



func (s *Server) loginSubmit(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))

	pass := r.FormValue("password")

	if !s.store.VerifyLogin(email, pass) {

		http.Error(w, "invalid credentials", http.StatusUnauthorized)

		return

	}

	u, ok := s.store.GetUserByEmail(email)

	if !ok {

		http.Error(w, "invalid credentials", http.StatusUnauthorized)

		return

	}

	code := uuid.NewString()

	s.mu.Lock()

	s.codes[code] = authCode{

		ClientID:    r.URL.Query().Get("client_id"),

		RedirectURI: r.URL.Query().Get("redirect_uri"),

		Challenge:   r.URL.Query().Get("code_challenge"),

		Method:      r.URL.Query().Get("code_challenge_method"),

		UserID:      u.ID,

		Expires:     time.Now().Add(5 * time.Minute),

	}

	s.mu.Unlock()

	redir, _ := url.Parse(r.URL.Query().Get("redirect_uri"))

	q := redir.Query()

	q.Set("code", code)

	if st := r.URL.Query().Get("state"); st != "" {

		q.Set("state", st)

	}

	redir.RawQuery = q.Encode()

	http.Redirect(w, r, redir.String(), http.StatusFound)

}



func (s *Server) token(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	if err := r.ParseForm(); err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)

		return

	}

	code := r.FormValue("code")

	verifier := r.FormValue("code_verifier")

	s.mu.RLock()

	ac, ok := s.codes[code]

	s.mu.RUnlock()

	if !ok || time.Now().After(ac.Expires) {

		http.Error(w, "invalid code", http.StatusBadRequest)

		return

	}

	if ac.Method == "S256" {

		h := sha256.Sum256([]byte(verifier))

		if base64.RawURLEncoding.EncodeToString(h[:]) != ac.Challenge {

			http.Error(w, "pkce failed", http.StatusBadRequest)

			return

		}

	}

	u, ok := s.store.GetUserByID(ac.UserID)

	if !ok {

		http.Error(w, "invalid user", http.StatusBadRequest)

		return

	}

	access, err := s.mintAccessToken(u)

	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return

	}

	s.mu.Lock()

	delete(s.codes, code)

	s.mu.Unlock()

	writeJSON(w, map[string]any{

		"access_token": access,

		"token_type":   "Bearer",

		"expires_in":   3600,

	})

}



// NewPKCEVerifier returns S256 verifier/challenge pair for clients.

func NewPKCEVerifier() (verifier, challenge string) {

	b := make([]byte, 32)

	_, _ = rand.Read(b)

	verifier = base64.RawURLEncoding.EncodeToString(b)

	h := sha256.Sum256([]byte(verifier))

	challenge = base64.RawURLEncoding.EncodeToString(h[:])

	return verifier, challenge

}



func writeJSON(w http.ResponseWriter, v any) {

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(v)

}


