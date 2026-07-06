package api

import (
	"encoding/json"
	"net/http"

	"era/services/vm/internal/cmdb"
)

// HandleSoftwareCVE — сверка installed software (CMDB) с продуктом для CVE-сканирования (ADR-0011).
func HandleSoftwareCVE(cp *cmdb.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		product := r.URL.Query().Get("product")
		if product == "" {
			http.Error(w, "product query required", http.StatusBadRequest)
			return
		}
		if cp == nil {
			http.Error(w, "ERA_CONTROL_PLANE_URL not set", http.StatusServiceUnavailable)
			return
		}
		rows, err := cp.ListSoftware()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		matches := cmdb.MatchProducts(rows, product)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"product": product,
			"matches": matches,
			"count":   len(matches),
		})
	}
}
