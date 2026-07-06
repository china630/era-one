// Package metrics — Prometheus exposition (GA-1 S5-22, GA-2 observability).
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "era_http_requests_total",
		Help: "HTTP requests by path and status",
	}, []string{"path", "method", "code"})

	EventsIngested = promauto.NewCounter(prometheus.CounterOpts{
		Name: "era_events_ingested_total",
		Help: "Events accepted by ingest-gateway",
	})

	EventsWritten = promauto.NewCounter(prometheus.CounterOpts{
		Name: "era_events_written_total",
		Help: "Events written to ClickHouse",
	})
)

func Handler() http.Handler {
	return promhttp.Handler()
}
