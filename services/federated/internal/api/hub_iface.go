package api

import "era/services/federated/internal/hub"

// HubAPI — общий интерфейс in-memory и persistent hub.
type HubAPI interface {
	Submit(sub hub.GradientSubmission) error
	Aggregate() ([]float64, int)
	GlobalModel() ([]float64, int)
	AuditEntries() []hub.SubmissionAudit
}
