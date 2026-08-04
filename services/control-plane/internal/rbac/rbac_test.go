package rbac

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFromRequestDevAcceptsClientAdmin(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "dev")
	t.Setenv("ERA_API_KEY", "")
	t.Setenv("ERA_AGENT_TOKEN", "")
	t.Setenv("ERA_PRODUCTION", "")
	t.Setenv("ERA_LICENSE_STRICT", "")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-ERA-Role", "admin")
	if FromRequest(req) != RoleAdmin {
		t.Fatal("dev must accept client admin")
	}
}

func TestFromRequestProxyRejectsSpoofAdmin(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "proxy")
	t.Setenv("ERA_API_KEY", "")
	t.Setenv("ERA_AGENT_TOKEN", "")
	t.Setenv("ERA_TRUSTED_PROXY_CIDRS", "")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-ERA-Role", "admin")
	if FromRequest(req) != RoleNone {
		t.Fatalf("spoof admin without Trusted-Proxy must be none, got %s", FromRequest(req))
	}
	// Header alone from untrusted peer must fail.
	req.Header.Set("X-ERA-Trusted-Proxy", "1")
	req.RemoteAddr = "203.0.113.50:443"
	if FromRequest(req) != RoleNone {
		t.Fatal("Trusted-Proxy from untrusted hop must be rejected")
	}
	req.RemoteAddr = "127.0.0.1:12345"
	if FromRequest(req) != RoleAdmin {
		t.Fatal("trusted proxy from loopback must accept admin")
	}
}

func TestFromRequestProxyTrustedCIDR(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "proxy")
	t.Setenv("ERA_API_KEY", "")
	t.Setenv("ERA_TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-ERA-Role", "admin")
	req.Header.Set("X-ERA-Trusted-Proxy", "1")
	req.RemoteAddr = "10.1.2.3:8080"
	if FromRequest(req) != RoleAdmin {
		t.Fatal("CIDR hop + Trusted-Proxy must accept admin")
	}
	req.RemoteAddr = "192.168.1.1:8080"
	if FromRequest(req) != RoleNone {
		t.Fatal("non-CIDR hop must be rejected")
	}
}

func TestFromRequestAPIKeyIgnoresHeaders(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "api_key")
	t.Setenv("ERA_API_KEY", "secret")
	t.Setenv("ERA_AGENT_TOKEN", "")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-ERA-Role", "admin")
	if FromRequest(req) != RoleNone {
		t.Fatal("api_key mode must ignore role header")
	}
	req.Header.Set("Authorization", "Bearer secret")
	if FromRequest(req) != RoleAdmin {
		t.Fatal("bearer must grant admin")
	}
}

func TestTrustedAgentSpoof(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "proxy")
	t.Setenv("ERA_API_KEY", "")
	t.Setenv("ERA_AGENT_TOKEN", "")
	t.Setenv("ERA_TRUSTED_PROXY_CIDRS", "")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-ERA-Actor", "era-agent")
	if IsTrustedAgent(req) {
		t.Fatal("spoof era-agent without proxy must fail")
	}
	req.Header.Set("X-ERA-Trusted-Proxy", "1")
	req.RemoteAddr = "198.51.100.9:9"
	if IsTrustedAgent(req) {
		t.Fatal("Trusted-Proxy from public IP must fail")
	}
	req.RemoteAddr = "127.0.0.1:9"
	if !IsTrustedAgent(req) {
		t.Fatal("trusted proxy + era-agent ok")
	}
}

func TestProductionDefaultsToProxy(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "")
	t.Setenv("ERA_API_KEY", "")
	t.Setenv("ERA_PRODUCTION", "1")
	if TrustFromEnv() != TrustProxy {
		t.Fatalf("got %s", TrustFromEnv())
	}
}

func TestAgentTokenIsNotAdmin(t *testing.T) {
	t.Setenv("ERA_RBAC_TRUST", "api_key")
	t.Setenv("ERA_API_KEY", "admin-secret")
	t.Setenv("ERA_AGENT_TOKEN", "agent-secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer agent-secret")
	if FromRequest(req) != RoleAgent {
		t.Fatalf("agent token must be RoleAgent, got %q", FromRequest(req))
	}
	if IsAdmin(FromRequest(req)) {
		t.Fatal("agent token must not be admin")
	}
	if !IsTrustedAgent(req) {
		t.Fatal("agent token must be trusted agent")
	}
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer admin-secret")
	if FromRequest(req2) != RoleAdmin {
		t.Fatal("API key must still grant admin")
	}
	if IsTrustedAgent(req2) {
		t.Fatal("API key alone is admin, not trusted-agent identity")
	}
}

func TestMiddlewareAllowsAgentTokenWithoutRole(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := Middleware(next)
	t.Setenv("ERA_RBAC_TRUST", "api_key")
	t.Setenv("ERA_API_KEY", "admin-k")
	t.Setenv("ERA_AGENT_TOKEN", "agent-k")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/enforcement/policy", nil)
	req.Header.Set("Authorization", "Bearer agent-k")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("agent token middleware want 200 got %d", rec.Code)
	}
}

func TestMiddlewareRejectsUnauthInProxyAndAPIKey(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := Middleware(next)

	t.Run("dev_allows", func(t *testing.T) {
		t.Setenv("ERA_RBAC_TRUST", "dev")
		t.Setenv("ERA_API_KEY", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("dev want 200 got %d", rec.Code)
		}
	})

	t.Run("proxy_spoof_role_401", func(t *testing.T) {
		t.Setenv("ERA_RBAC_TRUST", "proxy")
		t.Setenv("ERA_API_KEY", "")
		t.Setenv("ERA_TRUSTED_PROXY_CIDRS", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
		req.Header.Set("X-ERA-Role", "admin")
		req.RemoteAddr = "203.0.113.1:1"
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("spoof want 401 got %d", rec.Code)
		}
	})

	t.Run("proxy_trusted_hop_ok", func(t *testing.T) {
		t.Setenv("ERA_RBAC_TRUST", "proxy")
		t.Setenv("ERA_API_KEY", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
		req.Header.Set("X-ERA-Role", "admin")
		req.Header.Set("X-ERA-Trusted-Proxy", "1")
		req.RemoteAddr = "127.0.0.1:1"
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("trusted hop want 200 got %d", rec.Code)
		}
	})

	t.Run("api_key_spoof_401", func(t *testing.T) {
		t.Setenv("ERA_RBAC_TRUST", "api_key")
		t.Setenv("ERA_API_KEY", "k")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
		req.Header.Set("X-ERA-Role", "admin")
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("api_key spoof want 401 got %d", rec.Code)
		}
	})

	t.Run("api_key_bearer_ok", func(t *testing.T) {
		t.Setenv("ERA_RBAC_TRUST", "api_key")
		t.Setenv("ERA_API_KEY", "k")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
		req.Header.Set("Authorization", "Bearer k")
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("bearer want 200 got %d", rec.Code)
		}
	})

	t.Run("healthz_open", func(t *testing.T) {
		t.Setenv("ERA_RBAC_TRUST", "api_key")
		t.Setenv("ERA_API_KEY", "k")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("healthz want 200 got %d", rec.Code)
		}
	})
}
