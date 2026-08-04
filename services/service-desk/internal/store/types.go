package store

import "time"

// TicketStatus — общий статус ITIL-записи.
type TicketStatus string

const (
	StatusNew        TicketStatus = "new"
	StatusInProgress TicketStatus = "in_progress"
	StatusResolved   TicketStatus = "resolved"
	StatusClosed     TicketStatus = "closed"
)

// TicketKind — тип тикета для комментариев.
type TicketKind string

const (
	KindIncident TicketKind = "incident"
	KindRequest  TicketKind = "request"
	KindProblem  TicketKind = "problem"
	KindChange   TicketKind = "change"
)

// Incident — ITIL incident (MVP UI).
type Incident struct {
	ID          string       `json:"id"`
	TenantID    string       `json:"tenant_id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      TicketStatus `json:"status"`
	Priority    string       `json:"priority,omitempty"`
	NodeID      string       `json:"node_id,omitempty"`
	Requester   string       `json:"requester,omitempty"`
	Assignee    string       `json:"assignee,omitempty"`
	SLADueAt    *time.Time   `json:"sla_due_at,omitempty"`
	SLABreached bool         `json:"sla_breached,omitempty"`
	SLAStatus   string       `json:"sla_status,omitempty"` // none | ok | breached (computed)
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// ServiceRequest — ITIL service request (портал заявителя).
type ServiceRequest struct {
	ID        string       `json:"id"`
	TenantID  string       `json:"tenant_id"`
	Title     string       `json:"title"`
	Category  string       `json:"category,omitempty"`
	Status    TicketStatus `json:"status"`
	NodeID    string       `json:"node_id,omitempty"`
	Requester string       `json:"requester"`
	Assignee  string       `json:"assignee,omitempty"`
	SLAStatus string       `json:"sla_status,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// Problem — ITIL problem (схема с 1-го дня, UI позже).
type Problem struct {
	ID        string       `json:"id"`
	TenantID  string       `json:"tenant_id"`
	Title     string       `json:"title"`
	Status    TicketStatus `json:"status"`
	NodeID    string       `json:"node_id,omitempty"`
	Assignee  string       `json:"assignee,omitempty"`
	SLAStatus string       `json:"sla_status,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at,omitempty"`
}

// Change — ITIL change (схема с 1-го дня, UI позже).
type Change struct {
	ID        string       `json:"id"`
	TenantID  string       `json:"tenant_id"`
	Title     string       `json:"title"`
	Status    TicketStatus `json:"status"`
	Risk      string       `json:"risk,omitempty"`
	NodeID    string       `json:"node_id,omitempty"`
	Assignee  string       `json:"assignee,omitempty"`
	SLAStatus string       `json:"sla_status,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at,omitempty"`
}

// Comment — комментарий к тикету.
type Comment struct {
	ID        string     `json:"id"`
	TicketID  string     `json:"ticket_id"`
	Kind      TicketKind `json:"kind"`
	Author    string     `json:"author"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
}

// ComputeSLAStatus возвращает none|ok|breached для инцидента.
func ComputeSLAStatus(inc *Incident, now time.Time) string {
	if inc == nil || inc.SLADueAt == nil {
		return "none"
	}
	if inc.SLABreached || now.After(*inc.SLADueAt) {
		return "breached"
	}
	return "ok"
}

// EnrichIncident заполняет sla_status перед отдачей API.
func EnrichIncident(inc *Incident) *Incident {
	if inc == nil {
		return nil
	}
	inc.SLAStatus = ComputeSLAStatus(inc, time.Now().UTC())
	return inc
}
