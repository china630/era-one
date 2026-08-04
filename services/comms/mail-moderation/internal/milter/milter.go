// Package milter — минимальный stub Accept/Quarantine (CM-MM-11).
package milter

// Action — ответ milter-совместимого фильтра.
type Action string

const (
	ActionAccept     Action = "accept"
	ActionQuarantine Action = "quarantine"
	ActionReject     Action = "reject"
)

// Decision — результат stub-фильтра.
type Decision struct {
	Action Action
	Reason string
}

// Stub — мапит hold→quarantine, pass→accept.
func Stub(hold bool, reason string) Decision {
	if hold {
		return Decision{Action: ActionQuarantine, Reason: reason}
	}
	return Decision{Action: ActionAccept}
}
