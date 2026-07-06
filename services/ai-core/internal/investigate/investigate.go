// Package investigate — alert → storyline → verdict (F2-1, air-gap on-prem).
package investigate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Client struct {
	conn driver.Conn
	llm  LLM
}

type LLM interface {
	Complete(ctx context.Context, prompt string) (string, error)
	Available() bool
}

func New(addr, user, password string, llmClient LLM) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{Database: "era_xdr", Username: user, Password: password},
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(context.Background()); err != nil {
		return nil, err
	}
	return &Client{conn: conn, llm: llmClient}, nil
}

type Request struct {
	DetectionID string `json:"detection_id"`
	EventID     string `json:"event_id"`
	NodeID      string `json:"node_id"`
	TenantID    string `json:"tenant_id"`
}

type StoryStep struct {
	EventID    string `json:"event_id"`
	Category   string `json:"category"`
	ObservedAt string `json:"observed_at"`
	Summary    string `json:"summary"`
}

type Result struct {
	DetectionID string      `json:"detection_id"`
	Storyline   []StoryStep `json:"storyline"`
	Verdict     string      `json:"verdict"`
	Confidence  float64     `json:"confidence"`
	Narrative   string      `json:"narrative"`
	Mitre       []string    `json:"mitre_techniques"`
	CaseID      string      `json:"case_id,omitempty"`
}

func (c *Client) Investigate(ctx context.Context, req Request) (*Result, error) {
	if req.NodeID == "" {
		return nil, fmt.Errorf("node_id required")
	}
	tenant := req.TenantID
	if tenant == "" {
		tenant = "tenant-dev"
	}

	q := `SELECT event_id, category, observed_at, payload
		FROM era_xdr.events
		WHERE tenant_id = ? AND node_id = ?
		ORDER BY observed_at DESC LIMIT 50`
	rows, err := c.conn.Query(ctx, q, tenant, req.NodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []StoryStep
	suspicious := 0
	for rows.Next() {
		var eid, cat, payload string
		var obs time.Time
		if err := rows.Scan(&eid, &cat, &obs, &payload); err != nil {
			return nil, err
		}
		sum := summarize(cat, payload)
		if isSuspicious(payload) {
			suspicious++
		}
		steps = append(steps, StoryStep{
			EventID: eid, Category: cat,
			ObservedAt: obs.UTC().Format(time.RFC3339Nano),
			Summary:    sum,
		})
	}

	verdict := "benign"
	conf := 0.55
	if suspicious >= 3 {
		verdict = "malicious"
		conf = 0.92
	} else if suspicious >= 1 {
		verdict = "suspicious"
		conf = 0.78
	}

	narrative := fmt.Sprintf(
		"On-prem investigation for node %s: %d timeline events, %d suspicious indicators. ",
		req.NodeID, len(steps), suspicious,
	)
	switch verdict {
	case "malicious":
		narrative += "Multi-stage activity consistent with lateral movement (process → network → auth)."
	case "suspicious":
		narrative += "Anomalies detected; recommend analyst triage."
	default:
		narrative += "No strong malicious pattern in recent window."
	}

	mitre := inferMitre(steps, verdict)
	if c.llm != nil && c.llm.Available() {
		prompt := fmt.Sprintf("SOC analyst summary for node %s verdict %s events %d", req.NodeID, verdict, len(steps))
		if aiText, err := c.llm.Complete(ctx, prompt); err == nil && aiText != "" {
			narrative = strings.TrimSpace(narrative + " LLM: " + aiText)
		}
	}

	return &Result{
		DetectionID: req.DetectionID,
		Storyline:   steps,
		Verdict:     verdict,
		Confidence:  conf,
		Narrative:   narrative,
		Mitre:       mitre,
	}, nil
}

func inferMitre(steps []StoryStep, verdict string) []string {
	if verdict == "benign" {
		return nil
	}
	var out []string
	cats := map[string]bool{}
	for _, s := range steps {
		cats[s.Category] = true
	}
	if cats["process"] {
		out = append(out, "T1059")
	}
	if cats["network"] {
		out = append(out, "T1021")
	}
	if cats["auth"] {
		out = append(out, "T1078")
	}
	return out
}

func summarize(cat, payload string) string {
	var m map[string]any
	_ = json.Unmarshal([]byte(payload), &m)
	switch cat {
	case "process":
		return fmt.Sprintf("process %v", m["image_path"])
	case "network":
		return fmt.Sprintf("%v → %v:%v", m["src_ip"], m["dst_ip"], m["dst_port"])
	case "auth":
		return fmt.Sprintf("auth user=%v success=%v", m["user"], m["success"])
	case "file":
		return fmt.Sprintf("file %v", m["path"])
	default:
		if len(payload) > 80 {
			return payload[:80] + "…"
		}
		return payload
	}
}

func isSuspicious(payload string) bool {
	low := strings.ToLower(payload)
	for _, s := range []string{"powershell", "cmd.exe", "failed", "192.168.", "10.", "temp"} {
		if strings.Contains(low, s) {
			return true
		}
	}
	return false
}
