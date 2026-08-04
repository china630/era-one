package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"era/services/detection-engine/internal/chwriter"
	"era/services/detection-engine/internal/exposure"
	"era/services/detection-engine/internal/mitre"
	"era/services/detection-engine/internal/sigma"
	"era/services/platform/cpclient"
	"era/services/platform/metrics"
)

type ExposureServer struct {
	CH    *chwriter.Writer
	CP    *cpclient.Client
	Rules []*sigma.Rule
}

func (s *ExposureServer) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/api/v1/exposure", s.handleExposure)
	mux.HandleFunc("/api/v1/exposure/", s.handleExposureAsset)
	mux.HandleFunc("/api/v1/mitre/coverage", s.handleMitreCoverage)
	return mux
}

func (s *ExposureServer) handleMitreCoverage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cov := mitre.CorpusCoverage(s.Rules)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"techniques": cov,
		"source":     "corpus",
	})
}

func (s *ExposureServer) handleExposure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	topN := 10
	if v := r.URL.Query().Get("top"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			topN = n
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var detCounts, cveCounts map[string]map[string]int
	var err error
	if s.CH != nil {
		detCounts, err = s.CH.SeverityCountsByNode(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cveCounts, err = s.CH.VMFindingCountsByNode(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	meta := s.assetMeta()
	assets := exposure.BuildAssets(detCounts, cveCounts, meta)
	if aid := r.URL.Query().Get("asset_id"); aid != "" {
		for _, a := range assets {
			if a.NodeID == aid {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"asset": a})
				return
			}
		}
		// Fallback synthetic when CH empty (lab UI).
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"asset": map[string]any{
				"node_id": aid, "score": 0, "note": "no exposure rows yet",
			},
		})
		return
	}
	top := exposure.TopN(assets, topN)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"assets": assets,
		"top":    top,
	})
}

func (s *ExposureServer) handleExposureAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/exposure/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	q.Set("asset_id", id)
	r.URL.RawQuery = q.Encode()
	s.handleExposure(w, r)
}

func (s *ExposureServer) assetMeta() map[string]exposure.AssetMeta {
	out := make(map[string]exposure.AssetMeta)
	if s.CP == nil {
		return out
	}
	assets, err := s.CP.ListAssets()
	if err != nil {
		return out
	}
	for _, a := range assets {
		out[a.NodeID] = exposure.AssetMeta{
			Hostname: a.Hostname,
			Platform: a.Platform,
		}
	}
	return out
}
