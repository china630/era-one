// Package playbooks — SOAR плейбуки (F2-4, GA-1 real connectors).
package playbooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"era/services/soar/internal/connectors"

	"github.com/google/uuid"
)

type Action struct {
	ID        string    `json:"id"`
	Playbook  string    `json:"playbook"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

type Engine struct {
	mu      sync.Mutex
	actions []Action
	blocked map[string]bool

	ticket   *connectors.TicketWebhook
	isolator *connectors.HostIsolator
}

func New() *Engine {
	return NewFromEnv()
}

func NewFromEnv() *Engine {
	e := &Engine{blocked: make(map[string]bool)}
	if u := os.Getenv("ERA_SOAR_TICKET_WEBHOOK"); u != "" {
		e.ticket = connectors.NewTicketWebhook(u)
	}
	if p := os.Getenv("ERA_SOAR_ISOLATE_SCRIPT"); p != "" {
		e.isolator = &connectors.HostIsolator{ScriptPath: p}
	}
	return e
}

func (e *Engine) IsolateHost(nodeID string) Action {
	detail := fmt.Sprintf("host %s isolated (simulated firewall + agent quarantine)", nodeID)
	if e.isolator != nil {
		out, err := e.isolator.Isolate(nodeID)
		if err != nil {
			return e.recordFailed("isolate_host", err.Error())
		}
		detail = fmt.Sprintf("host %s isolated via script: %s", nodeID, out)
	}
	return e.record("isolate_host", detail)
}

func (e *Engine) BlockIP(ip string) Action {
	e.mu.Lock()
	e.blocked[ip] = true
	e.mu.Unlock()
	detail := fmt.Sprintf("ip %s blocked at perimeter (simulated)", ip)
	if u := os.Getenv("ERA_SOAR_BLOCK_IP_WEBHOOK"); u != "" {
		wh := connectors.NewTicketWebhook(u)
		if _, err := wh.Create("block_ip:"+ip, ""); err == nil {
			detail = fmt.Sprintf("ip %s block requested via webhook", ip)
		}
	}
	return e.record("block_ip", detail)
}

func (e *Engine) CreateTicket(title, caseID string) Action {
	if e.ticket != nil {
		res, err := e.ticket.Create(title, caseID)
		if err != nil {
			return e.recordFailed("create_ticket", err.Error())
		}
		ext := res.ExternalID
		if ext == "" {
			ext = uuid.NewString()
		}
		e.linkCase(caseID, ext)
		return e.record("create_ticket", fmt.Sprintf("ticket %s for case %s: %s (%s)", ext, caseID, title, res.Detail))
	}
	tid := uuid.NewString()
	return e.record("create_ticket", fmt.Sprintf("ticket %s created for case %s: %s (simulated)", tid, caseID, title))
}

func (e *Engine) IsBlocked(ip string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.blocked[ip]
}

func (e *Engine) Actions() []Action {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Action, len(e.actions))
	copy(out, e.actions)
	return out
}

func (e *Engine) record(playbook, detail string) Action {
	e.mu.Lock()
	defer e.mu.Unlock()
	a := Action{
		ID: uuid.NewString(), Playbook: playbook, Status: "completed",
		Detail: detail, CreatedAt: time.Now().UTC(),
	}
	e.actions = append(e.actions, a)
	return a
}

func (e *Engine) recordFailed(playbook, detail string) Action {
	e.mu.Lock()
	defer e.mu.Unlock()
	a := Action{
		ID: uuid.NewString(), Playbook: playbook, Status: "failed",
		Detail: detail, CreatedAt: time.Now().UTC(),
	}
	e.actions = append(e.actions, a)
	return a
}

func (e *Engine) linkCase(caseID, ticketID string) {
	base := os.Getenv("ERA_CONTROL_PLANE_URL")
	if base == "" || caseID == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{
		"body": fmt.Sprintf("SOAR ticket linked: %s", ticketID),
	})
	req, err := http.NewRequest(http.MethodPost, base+"/api/v1/cases/"+caseID+"/notes", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ERA-Actor", "soar")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
