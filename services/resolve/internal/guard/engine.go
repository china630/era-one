package guard

import (
	"era/services/resolve/internal/atlas"
	"era/services/resolve/internal/policy"
)

// Verdict is a Guard decision for one query.
type Verdict struct {
	QName    string        `json:"qname"`
	QType    string        `json:"qtype"`
	Action   policy.Action `json:"action"`
	RuleID   string        `json:"rule_id,omitempty"`
	Reason   string        `json:"reason,omitempty"`
	Sinkhole string        `json:"sinkhole_ip,omitempty"`
	Source   string        `json:"source"` // policy | atlas | default
}

// Engine combines Atlas + policy.
type Engine struct {
	Policy *policy.Store
	Atlas  *atlas.Store
	DefaultSinkhole string
}

func New(pol *policy.Store, atl *atlas.Store) *Engine {
	return &Engine{Policy: pol, Atlas: atl, DefaultSinkhole: "127.0.0.1"}
}

// Decide returns allow/nxdomain/sinkhole for qname.
func (e *Engine) Decide(qname, qtype string) Verdict {
	v := Verdict{QName: qname, QType: qtype, Action: policy.ActionAllow, Source: "default"}
	if e.Atlas != nil {
		if hit, ok := e.Atlas.Lookup(qname); ok {
			v.Action = policy.ActionNXDomain
			v.RuleID = "atlas:" + hit.Domain
			v.Reason = "atlas hit severity=" + hit.Severity
			v.Source = "atlas"
			return v
		}
	}
	if e.Policy != nil {
		if rule, ok := e.Policy.Match(qname); ok {
			v.Action = rule.Action
			v.RuleID = rule.ID
			v.Reason = rule.Reason
			v.Source = "policy"
			if rule.Action == policy.ActionSinkhole {
				v.Sinkhole = rule.SinkholeIP
				if v.Sinkhole == "" {
					v.Sinkhole = e.DefaultSinkhole
				}
			}
			return v
		}
	}
	return v
}
