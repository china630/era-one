// Package engine — связка policy → resolve → hold → notify → audit.
package engine

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"era/services/comms/mail-moderation/internal/audit"
	"era/services/comms/mail-moderation/internal/hold"
	"era/services/comms/mail-moderation/internal/notify"
	"era/services/comms/mail-moderation/internal/policy"
	"era/services/comms/mail-moderation/internal/resolve"
)

// GroupLookup — группы отправителя (AD/local).
type GroupLookup interface {
	Groups(sender string) []string
}

// StaticGroups — map sender → groups.
type StaticGroups map[string][]string

func (s StaticGroups) Groups(sender string) []string {
	return s[strings.ToLower(extractAddr(sender))]
}

// Upstream — доставка после approve / pass.
type Upstream interface {
	Deliver(raw []byte, from string, to []string) error
}

// MemoryUpstream записывает доставки.
type MemoryUpstream struct {
	Delivered []Delivery
}

type Delivery struct {
	From string
	To   []string
	Raw  []byte
}

func (m *MemoryUpstream) Deliver(raw []byte, from string, to []string) error {
	m.Delivered = append(m.Delivered, Delivery{From: from, To: append([]string(nil), to...), Raw: append([]byte(nil), raw...)})
	return nil
}

// Engine — ядро модерации.
type Engine struct {
	Rules    []policy.Rule
	Local    []string
	Groups   GroupLookup
	Resolve  *resolve.Resolver
	Holds    hold.Repository
	Notify   *notify.Service
	Audit    audit.Recorder
	Upstream Upstream
}

// ProcessRaw обрабатывает RFC822 / простой raw после SMTP DATA.
func (e *Engine) ProcessRaw(raw []byte, envelopeFrom string, envelopeTo []string) (policy.Decision, string, error) {
	msg := parseMessage(raw, envelopeFrom, envelopeTo)
	if e.Groups != nil {
		msg.SenderGroups = e.Groups.Groups(msg.From)
	}
	res := policy.Evaluate(e.Rules, msg, policy.EvalContext{LocalDomains: e.Local})
	switch res.Decision {
	case policy.DecisionPass, policy.DecisionBypass:
		if e.Upstream != nil {
			if err := e.Upstream.Deliver(raw, msg.From, msg.To); err != nil {
				return res.Decision, "", err
			}
		}
		return res.Decision, "", nil
	case policy.DecisionHold:
		mods, err := e.Resolve.Resolve(extractAddr(msg.From), res.Rule.Moderator)
		if err != nil {
			return res.Decision, "", err
		}
		rec, err := e.Holds.Put(hold.Record{
			RuleID:     res.RuleID,
			Sender:     extractAddr(msg.From),
			Recipients: msg.To,
			Subject:    msg.Subject,
			Moderators: mods,
			Raw:        raw,
			ExpiresAt:  time.Now().UTC().Add(policy.EffectiveTTL(res.Rule)),
		})
		if err != nil {
			return res.Decision, "", err
		}
		_ = e.Audit.Record(audit.Event{HoldID: rec.ID, Action: "hold", Sender: rec.Sender, RuleID: res.RuleID})
		if e.Notify != nil {
			_ = e.Notify.NotifyModerator(rec.ID, rec.Sender, rec.Subject, mods, truncate(string(raw), 500))
			if policy.NotifyOnHoldDefault(res.Rule) {
				_ = e.Notify.NotifySenderHeld(rec.Sender, rec.Subject, rec.ID)
			}
		}
		return policy.DecisionHold, rec.ID, nil
	default:
		return res.Decision, "", fmt.Errorf("unknown decision")
	}
}

// ApplyAction approve/reject по hold id.
func (e *Engine) ApplyAction(holdID, moderator, action, comment string) error {
	action = strings.ToLower(action)
	var rec hold.Record
	var err error
	switch action {
	case "approve":
		rec, err = e.Holds.Approve(holdID, moderator)
	case "reject":
		rec, err = e.Holds.Reject(holdID, moderator, comment)
	default:
		return fmt.Errorf("bad action %q", action)
	}
	if err != nil {
		return err
	}
	_ = e.Audit.Record(audit.Event{
		HoldID: holdID, Action: action, Sender: rec.Sender, RuleID: rec.RuleID, Moderator: moderator,
		Meta: map[string]string{"comment": comment},
	})
	if action == "approve" && e.Upstream != nil {
		return e.Upstream.Deliver(rec.Raw, rec.Sender, rec.Recipients)
	}
	if action == "reject" && e.Notify != nil {
		return e.Notify.NotifySenderRejected(rec.Sender, rec.Subject, comment)
	}
	return nil
}

// RunTTL — auto-reject (default) expired holds.
func (e *Engine) RunTTL() {
	expired := e.Holds.ExpirePending(false)
	for _, rec := range expired {
		_ = e.Audit.Record(audit.Event{HoldID: rec.ID, Action: "expire", Sender: rec.Sender, RuleID: rec.RuleID})
		if e.Notify != nil {
			_ = e.Notify.NotifySenderRejected(rec.Sender, rec.Subject, "ttl expired")
		}
	}
}

func parseMessage(raw []byte, envFrom string, envTo []string) policy.Message {
	msg := policy.Message{
		From: envFrom,
		To:   append([]string(nil), envTo...),
	}
	text := string(raw)
	if m, err := mail.ReadMessage(strings.NewReader(text)); err == nil {
		msg.From = firstNonEmpty(m.Header.Get("From"), envFrom)
		msg.Subject = m.Header.Get("Subject")
		if len(envTo) == 0 {
			if to := m.Header.Get("To"); to != "" {
				msg.To = []string{to}
			}
		}
	} else {
		// crude subject
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(strings.ToLower(line), "subject:") {
				msg.Subject = strings.TrimSpace(line[8:])
				break
			}
		}
	}
	msg.Body = text
	msg.IsSystem = strings.Contains(strings.ToLower(msg.From), "mailer-daemon")
	msg.Authenticated = true
	return msg
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func extractAddr(s string) string {
	s = strings.TrimSpace(s)
	if a, err := mail.ParseAddress(s); err == nil {
		return strings.ToLower(a.Address)
	}
	return strings.ToLower(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
