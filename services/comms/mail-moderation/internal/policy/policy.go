// Package policy — правила outbound moderation (AC-MM-1,3,4,6,10).
package policy

import (
	"fmt"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Decision — исход оценки письма.
type Decision string

const (
	DecisionPass   Decision = "pass"
	DecisionHold   Decision = "hold"
	DecisionBypass Decision = "bypass"
)

// ModeratorMode — как выбрать модератора.
type ModeratorMode string

const (
	ModManager     ModeratorMode = "manager"
	ModStatic      ModeratorMode = "static"
	ModLDAPAttr    ModeratorMode = "ldap_attr"
	ModCuratorMap  ModeratorMode = "curator_map"
)

// Conditions — условия срабатывания правила.
type Conditions struct {
	SenderGroups    []string `yaml:"sender_groups"`
	ExternalOnly    bool     `yaml:"external_only"`
	InternalAlso    bool     `yaml:"internal_also"`
	Keywords        []string `yaml:"keywords"`
	KeywordRegex    []string `yaml:"keyword_regex"`
	RecipientAddrs  []string `yaml:"recipient_addrs"`
	RecipientDomains []string `yaml:"recipient_domains"`
	HasAttachment   *bool    `yaml:"has_attachment"`
	MaxAttachmentKB int      `yaml:"max_attachment_kb"`
	AttachmentExtAllow []string `yaml:"attachment_ext_allow"`
	AttachmentExtDeny  []string `yaml:"attachment_ext_deny"`
	BypassSenders   []string `yaml:"bypass_senders"`
	BypassGroups    []string `yaml:"bypass_groups"`
	ExcludeSystem   bool     `yaml:"exclude_system"` // NDR / mailer-daemon
	// ModeratedRecipients — P1: hold if any envelope recipient matches (DL / mailbox).
	ModeratedRecipients []string `yaml:"moderated_recipients"`
}

// ModeratorSpec — резолв модератора для правила.
type ModeratorSpec struct {
	Mode       ModeratorMode `yaml:"mode"`
	Static     []string      `yaml:"static"`
	LDAPAttr   string        `yaml:"ldap_attr"`
	Fallback   []string      `yaml:"fallback"`
}

// Rule — одно транспортное правило модерации.
type Rule struct {
	ID               string        `yaml:"id"`
	Priority         int           `yaml:"priority"`
	StopProcessing   bool          `yaml:"stop_processing"`
	Conditions       Conditions    `yaml:"conditions"`
	Moderator        ModeratorSpec `yaml:"moderator"`
	TTLHours         int           `yaml:"ttl_hours"`
	NotifyOnHold     *bool         `yaml:"notify_on_hold"`
	NotifyOnApprove  *bool         `yaml:"notify_on_approve"`
	AutoApproveOnTTL bool          `yaml:"auto_approve_on_ttl"`
}

// Document — YAML-файл набора правил.
type Document struct {
	Rules []Rule `yaml:"rules"`
}

// Message — минимальный контекст письма для Evaluate.
type Message struct {
	From            string
	To              []string
	Subject         string
	Body            string
	HasAttachment   bool
	AttachmentExts  []string
	AttachmentSizeKB int
	SenderGroups    []string
	IsSystem        bool // NDR / mailer-daemon
	Authenticated   bool
}

// EvalContext — каталог/орг для условий (группы уже в Message.SenderGroups).
type EvalContext struct {
	LocalDomains []string
}

// Result — решение + сработавшее правило.
type Result struct {
	Decision Decision
	RuleID   string
	Rule     *Rule
}

// Evaluate применяет правила по возрастанию priority; первое match + stop.
func Evaluate(rules []Rule, msg Message, ctx EvalContext) Result {
	sorted := append([]Rule(nil), rules...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	for i := range sorted {
		r := &sorted[i]
		if matchBypass(r.Conditions, msg) {
			if r.StopProcessing {
				return Result{Decision: DecisionBypass, RuleID: r.ID, Rule: r}
			}
			continue
		}
		if !matchConditions(r.Conditions, msg, ctx) {
			continue
		}
		res := Result{Decision: DecisionHold, RuleID: r.ID, Rule: r}
		if r.StopProcessing || true {
			// Первое совпадение всегда останавливает (stop_processing default behavior for hold).
			return res
		}
	}
	return Result{Decision: DecisionPass}
}

func matchBypass(c Conditions, msg Message) bool {
	from := strings.ToLower(extractAddr(msg.From))
	for _, s := range c.BypassSenders {
		if strings.EqualFold(extractAddr(s), from) {
			return true
		}
	}
	for _, g := range c.BypassGroups {
		for _, sg := range msg.SenderGroups {
			if strings.EqualFold(g, sg) {
				return true
			}
		}
	}
	return false
}

func matchConditions(c Conditions, msg Message, ctx EvalContext) bool {
	if c.ExcludeSystem && msg.IsSystem {
		return false
	}
	if len(c.SenderGroups) > 0 && !intersectsFold(c.SenderGroups, msg.SenderGroups) {
		return false
	}
	external := isExternal(msg.To, ctx.LocalDomains)
	if c.ExternalOnly && !c.InternalAlso && !external {
		return false
	}
	if !c.ExternalOnly && !c.InternalAlso && !external {
		// no direction constraint
	}
	if c.InternalAlso {
		// allow both; ExternalOnly+InternalAlso = all mail matching other filters
	} else if c.ExternalOnly && !external {
		return false
	}
	if len(c.Keywords) > 0 || len(c.KeywordRegex) > 0 {
		blob := msg.Subject + "\n" + msg.Body
		if !matchKeywords(blob, c.Keywords, c.KeywordRegex) {
			return false
		}
	}
	if len(c.RecipientAddrs) > 0 || len(c.RecipientDomains) > 0 {
		if !matchRecipients(msg.To, c.RecipientAddrs, c.RecipientDomains) {
			return false
		}
	}
	if len(c.ModeratedRecipients) > 0 {
		if !matchRecipients(msg.To, c.ModeratedRecipients, nil) {
			return false
		}
	}
	if c.HasAttachment != nil && msg.HasAttachment != *c.HasAttachment {
		return false
	}
	if c.MaxAttachmentKB > 0 && msg.AttachmentSizeKB > c.MaxAttachmentKB {
		return false
	}
	if len(c.AttachmentExtDeny) > 0 {
		for _, ext := range msg.AttachmentExts {
			for _, d := range c.AttachmentExtDeny {
				if strings.EqualFold(ext, d) {
					return true // deny ext still matches moderation (hold)
				}
			}
		}
	}
	if len(c.AttachmentExtAllow) > 0 && msg.HasAttachment {
		ok := false
		for _, ext := range msg.AttachmentExts {
			for _, a := range c.AttachmentExtAllow {
				if strings.EqualFold(ext, a) {
					ok = true
				}
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func matchKeywords(blob string, kws, regs []string) bool {
	lower := strings.ToLower(blob)
	for _, k := range kws {
		if k != "" && strings.Contains(lower, strings.ToLower(k)) {
			return true
		}
	}
	for _, r := range regs {
		re, err := regexp.Compile("(?i)" + r)
		if err != nil {
			continue
		}
		if re.MatchString(blob) {
			return true
		}
	}
	return len(kws) == 0 && len(regs) == 0
}

func matchRecipients(to, addrs, domains []string) bool {
	for _, t := range to {
		a := strings.ToLower(extractAddr(t))
		for _, want := range addrs {
			if a == strings.ToLower(extractAddr(want)) {
				return true
			}
		}
		_, domain, _ := strings.Cut(a, "@")
		for _, d := range domains {
			if strings.EqualFold(domain, d) {
				return true
			}
		}
	}
	return false
}

func isExternal(to []string, local []string) bool {
	if len(to) == 0 {
		return false
	}
	for _, t := range to {
		_, domain, ok := strings.Cut(strings.ToLower(extractAddr(t)), "@")
		if !ok {
			continue
		}
		localHit := false
		for _, ld := range local {
			if strings.EqualFold(domain, ld) {
				localHit = true
				break
			}
		}
		if !localHit {
			return true
		}
	}
	return false
}

func intersectsFold(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if strings.EqualFold(x, y) {
				return true
			}
		}
	}
	return false
}

func extractAddr(s string) string {
	s = strings.TrimSpace(s)
	if a, err := mail.ParseAddress(s); err == nil {
		return a.Address
	}
	return s
}

// EffectiveTTL возвращает TTL правила или default 72h.
func EffectiveTTL(r *Rule) time.Duration {
	if r == nil || r.TTLHours <= 0 {
		return 72 * time.Hour
	}
	return time.Duration(r.TTLHours) * time.Hour
}

// NotifyOnHoldDefault — default ON для шаблона новичков.
func NotifyOnHoldDefault(r *Rule) bool {
	if r == nil || r.NotifyOnHold == nil {
		return true
	}
	return *r.NotifyOnHold
}

// ValidateDocument проверяет уникальность id.
func ValidateDocument(doc Document) error {
	seen := map[string]struct{}{}
	for _, r := range doc.Rules {
		if r.ID == "" {
			return fmt.Errorf("rule missing id")
		}
		if _, ok := seen[r.ID]; ok {
			return fmt.Errorf("duplicate rule id %q", r.ID)
		}
		seen[r.ID] = struct{}{}
	}
	return nil
}
