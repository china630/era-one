package api

import (
	"encoding/json"
	"log"
	"net/http"

	"era/services/vm/internal/models"
	"era/services/vm/internal/publisher"
	"era/services/vm/internal/scanner"
)

// ScanRequest описывает входной JSON-запрос на запуск сканирования.
type ScanRequest struct {
	Targets     []string `json:"targets"`
	Concurrency int      `json:"concurrency"`
}

// ScanResponse описывает JSON-ответ с результатами сканирования.
type ScanResponse struct {
	Status   string           `json:"status"`
	Findings []models.Finding `json:"findings"`
}

// HandleScan возвращает HTTP-хэндлер запуска сканирования.
func HandleScan(engine *scanner.Engine, pub *publisher.Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		findings := engine.Run(req.Targets)
		if pub != nil && len(findings) > 0 {
			if err := pub.PublishFindings(r.Context(), findings); err != nil {
				log.Printf("kafka publish findings: %v", err)
			}
		}
		resp := ScanResponse{
			Status:   "ok",
			Findings: findings,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
