package proxy

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"era/services/comms/mail-bridge/internal/audit"
)

// CalDAV proxies calendar requests to upstream CalDAV (IceWarp/CG).
type CalDAV struct {
	UpstreamBase string
	Audit        *audit.Recorder
}

func CalDAVFromEnv() *CalDAV {
	return &CalDAV{UpstreamBase: os.Getenv("ERA_BRIDGE_UPSTREAM_CALDAV_BASE_URL")}
}

func (p *CalDAV) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.UpstreamBase == "" {
		http.Error(w, "caldav upstream not configured", http.StatusBadGateway)
		return
	}
	target, err := url.Parse(strings.TrimRight(p.UpstreamBase, "/"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	rp.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
	if p.Audit != nil {
		p.Audit.Record("BRIDGE_CALDAV_"+r.Method, r.URL.Path)
	}
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/caldav")
	r.URL.Host = target.Host
	r.URL.Scheme = target.Scheme
	r.Host = target.Host
	rp.ServeHTTP(w, r)
}

// Ensure CalDAV implements http.Handler
var _ http.Handler = (*CalDAV)(nil)

// Drain for compile
var _ = io.EOF
