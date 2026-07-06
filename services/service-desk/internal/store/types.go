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
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// ServiceRequest — ITIL service request (портал заявителя).
type ServiceRequest struct {
	ID          string       `json:"id"`
	TenantID    string       `json:"tenant_id"`
	Title       string       `json:"title"`
	Category    string       `json:"category,omitempty"`
	Status      TicketStatus `json:"status"`
	NodeID      string       `json:"node_id,omitempty"`
	Requester   string       `json:"requester"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Problem — ITIL problem (схема с 1-го дня, UI позже).
type Problem struct {
	ID        string       `json:"id"`
	TenantID  string       `json:"tenant_id"`
	Title     string       `json:"title"`
	Status    TicketStatus `json:"status"`
	NodeID    string       `json:"node_id,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// Change — ITIL change (схема с 1-го дня, UI позже).
type Change struct {
	ID        string       `json:"id"`
	TenantID  string       `json:"tenant_id"`
	Title     string       `json:"title"`
	Status    TicketStatus `json:"status"`
	Risk      string       `json:"risk,omitempty"`
	NodeID    string       `json:"node_id,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}
