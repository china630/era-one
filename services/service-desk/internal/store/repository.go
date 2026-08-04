package store

import (
	"fmt"
	"os"
	"time"
)

// Repository — ITSM-lite store (memory MVP).
type Repository interface {
	CreateIncident(i *Incident)
	GetIncident(id string) (*Incident, bool)
	UpdateIncident(id string, fn func(*Incident)) (*Incident, bool)
	ListIncidents() []*Incident
	CreateRequest(r *ServiceRequest)
	GetRequest(id string) (*ServiceRequest, bool)
	UpdateRequest(id string, fn func(*ServiceRequest)) (*ServiceRequest, bool)
	ListRequests() []*ServiceRequest
	CreateProblem(p *Problem)
	GetProblem(id string) (*Problem, bool)
	UpdateProblem(id string, fn func(*Problem)) (*Problem, bool)
	ListProblems() []*Problem
	CreateChange(c *Change)
	GetChange(id string) (*Change, bool)
	UpdateChange(id string, fn func(*Change)) (*Change, bool)
	ListChanges() []*Change
	AddComment(c *Comment)
	ListComments(kind TicketKind, ticketID string) []*Comment
}

func nowUTC() time.Time { return time.Now().UTC() }

// NewFromEnv — memory (default) или sqlite при ERA_STORE_DRIVER=sqlite.
func NewFromEnv() (Repository, error) {
	switch os.Getenv("ERA_STORE_DRIVER") {
	case "sqlite":
		path := os.Getenv("ERA_STORE_PATH")
		if path == "" {
			path = "service-desk.db"
		}
		return NewSQLite(path)
	default:
		return NewMemory(), nil
	}
}

// CloseableRepository — store с Close().
type CloseableRepository interface {
	Repository
	Close() error
}

func closeIfNeeded(st Repository) {
	if c, ok := st.(CloseableRepository); ok {
		_ = c.Close()
	}
}

func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	return nil
}
