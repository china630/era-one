package sla

import (
	"time"

	"era/services/service-desk/internal/store"
)

// Engine — проверка SLA breach и эскалация приоритета.
type Engine struct {
	Store store.Repository
	Now   func() time.Time
}

func NewEngine(st store.Repository) *Engine {
	return &Engine{Store: st, Now: time.Now}
}

// CheckBreaches помечает просроченные инциденты и эскалирует priority.
func (e *Engine) CheckBreaches() []string {
	now := e.Now().UTC()
	var breached []string
	for _, inc := range e.Store.ListIncidents() {
		if inc.SLADueAt == nil || inc.SLABreached {
			continue
		}
		if inc.Status == store.StatusClosed || inc.Status == store.StatusResolved {
			continue
		}
		if now.After(*inc.SLADueAt) {
			id := inc.ID
			_, _ = e.Store.UpdateIncident(id, func(i *store.Incident) {
				i.SLABreached = true
				if i.Priority == "" || i.Priority == "low" {
					i.Priority = "high"
				} else if i.Priority == "medium" {
					i.Priority = "critical"
				}
			})
			breached = append(breached, id)
		}
	}
	return breached
}
