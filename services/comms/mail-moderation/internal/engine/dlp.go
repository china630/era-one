package engine

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// DLPResult — honesty mode for Perimeter handoff (G0-7).
type DLPResult struct {
	Mode   string // live | stub
	Status int
}

// DLPHandoff POSTs hold metadata to ERA_MM_DLP_URL; unset URL → mode=stub (not silent success).
func DLPHandoff(holdID, sender, subject, ruleID string) DLPResult {
	url := strings.TrimSpace(os.Getenv("ERA_MM_DLP_URL"))
	if url == "" {
		return DLPResult{Mode: "stub", Status: 0}
	}
	body, _ := json.Marshal(map[string]string{
		"hold_id": holdID,
		"sender":  sender,
		"subject": subject,
		"rule_id": ruleID,
		"source":  "era-mail-moderation",
		"mode":    "live",
	})
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return DLPResult{Mode: "stub", Status: 0}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return DLPResult{Mode: "stub", Status: 0}
	}
	defer resp.Body.Close()
	return DLPResult{Mode: "live", Status: resp.StatusCode}
}
