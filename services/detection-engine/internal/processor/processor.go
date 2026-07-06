// Package processor — Sigma + корреляция по потоку событий.

package processor



import (

	"context"

	"encoding/json"

	"log"

	"time"



	erav1 "era/contracts/gen/era/v1"

	"github.com/google/uuid"

	"github.com/oklog/ulid"

	"era/services/detection-engine/internal/chwriter"

	"era/services/detection-engine/internal/correlator"

	"era/services/detection-engine/internal/itdr"

	"era/services/detection-engine/internal/ndr"

	"era/services/detection-engine/internal/risk"

	"era/services/detection-engine/internal/sigma"

	"era/services/detection-engine/internal/tip"

	"era/services/platform/cpclient"

)



type Processor struct {

	Rules      []*sigma.Rule

	Corr       *correlator.Engine

	NDR        *ndr.Engine

	National   *tip.Feed

	STIX       *tip.Feed

	Risk       *risk.Scorer

	Detections detectionWriter
	CP         *cpclient.Client
	caseDedup  map[string]time.Time
}

type detectionWriter interface {
	InsertDetection(ctx context.Context, d chwriter.DetectionRow) error
}

func New(rules []*sigma.Rule, w detectionWriter, national, stix *tip.Feed, cp *cpclient.Client) *Processor {
	return &Processor{
		Rules:      rules,
		Corr:       correlator.New(10 * time.Minute),
		NDR:        ndr.New(15 * time.Minute),
		National:   national,
		STIX:       stix,
		Risk:       risk.New(15 * time.Minute),
		Detections: w,
		CP:         cp,
		caseDedup:  make(map[string]time.Time),
	}
}



func (p *Processor) Handle(ctx context.Context, env *erav1.Envelope) {

	if env == nil {

		return

	}

	payload := payloadJSON(env)

	cat := correlator.CategoryName(env.GetCategory())

	node := ""

	if s := env.GetSource(); s != nil {

		node = s.GetNodeId()

	}

	obs := time.Now().UTC()

	if t := env.GetObservedAt(); t != nil {

		obs = t.AsTime().UTC()

	}

	eid := eventIDStr(env.GetEventId())



	p.Corr.Observe(node, cat, eid, payload, obs)



	switch cat {

	case "network":

		srcIP, dstIP, dstPort := parseNetwork(payload)

		if srcIP != "" {

			p.NDR.Observe(srcIP, dstIP, dstPort, obs)

			p.NDR.ObserveBeacon(srcIP, dstIP, obs)

			bytesOut, ja3 := ndr.ParseNetworkExfil(payload)
			if bytesOut > 0 || ja3 != "" {
				p.NDR.ObserveExfil(srcIP, bytesOut, ja3, obs)
			}

			if ok, ruleID := p.NDR.LateralMovement(srcIP); ok {

				p.emit(ctx, env, ruleID, "NDR lateral movement T1021", "critical", "ndr", eid, obs, node)

			}

			if ok, ruleID := p.NDR.Beaconing(srcIP, dstIP); ok {

				p.emit(ctx, env, ruleID, "NDR C2 beaconing T1071", "high", "ndr", eid, obs, node)

			}

			if ok, ruleID := p.NDR.DataExfil(srcIP); ok {

				p.emit(ctx, env, ruleID, "NDR data exfiltration T1048", "high", "ndr", eid, obs, node)

			}

			if ok, ruleID := p.NDR.JA3Fingerprint(srcIP); ok {

				p.emit(ctx, env, ruleID, "NDR malicious JA3 fingerprint", "high", "ndr", eid, obs, node)

			}

		}

	case "dns":

		srcIP, query := ndr.ParseDNS(payload)

		if srcIP != "" && query != "" {

			p.NDR.ObserveDNS(srcIP, query, obs)

			if ok, ruleID := p.NDR.DNSTunnel(srcIP); ok {

				p.emit(ctx, env, ruleID, "NDR DNS tunnel T1071.004", "high", "ndr", eid, obs, node)

			}

		}

	case "auth":

		if ok, rule := itdr.MatchAuth(payload); ok {

			p.emit(ctx, env, rule.ID, rule.Title, rule.Level, "itdr", eid, obs, node)

		}

	}



	for _, rule := range p.Rules {

		if rule.Match(cat, payload) {

			p.emit(ctx, env, rule.ID, rule.Title, rule.Level, "sigma", eid, obs, node)

		}

	}



	if p.National != nil {

		if ok, ruleID := p.National.Match(payload); ok {

			p.emit(ctx, env, ruleID, "National IOC match", "high", "national-tip", eid, obs, node)

		}

	}

	if p.STIX != nil {

		if ok, ruleID := p.STIX.Match(payload); ok {

			p.emit(ctx, env, ruleID, "STIX IOC match", "high", "stix-tip", eid, obs, node)

		}

	}



	if ok, ruleID := p.Corr.APTChain(node); ok {

		p.emit(ctx, env, ruleID, "APT lateral movement chain", "high", "correlation", eid, obs, node)

	}

	if ok, ruleID := p.Corr.ObserveNetworkEndpoint(node); ok {

		p.emit(ctx, env, ruleID, "Observe network alert + suspicious endpoint", "high", "correlation", eid, obs, node)

	}

}



func (p *Processor) emit(ctx context.Context, env *erav1.Envelope, ruleID, ruleName, level, engine, eventID string, obs time.Time, node string) {

	if p.Risk != nil && !p.Risk.ShouldEmit(ruleID, node, obs) {

		return

	}

	if p.Risk != nil {

		score := p.Risk.Bump(node, level, obs)

		if score >= 50 && level != "critical" {

			level = "high"

		}

	}

	did := uuid.NewString()

	tenant := ""

	if s := env.GetSource(); s != nil {

		tenant = s.GetTenantId()

	}

	if err := p.Detections.InsertDetection(ctx, chwriter.DetectionRow{

		DetectionID: did,

		EventID:     eventID,

		ObservedAt:  obs,

		TenantID:    tenant,

		NodeID:      node,

		RuleID:      ruleID,

		RuleName:    ruleName,

		Severity:    level,

		Engine:      engine,

		Confidence:  0.85,

	}); err != nil {

		log.Printf("detection insert: %v", err)

		return

	}

	log.Printf("DETECTION rule=%s event=%s node=%s", ruleID, eventID, node)
	p.maybeAutoCase(ruleID, ruleName, level, node, obs)
}

func (p *Processor) maybeAutoCase(ruleID, ruleName, level, node string, obs time.Time) {
	if p.CP == nil || node == "" {
		return
	}
	if level != "high" && level != "critical" {
		return
	}
	key := ruleID + "|" + node
	if last, ok := p.caseDedup[key]; ok && obs.Sub(last) < 15*time.Minute {
		return
	}
	title := "Detection: " + ruleName + " on " + node
	if _, err := p.CP.CreateCase(title, ruleID, node); err != nil {
		log.Printf("auto-case: %v", err)
		return
	}
	p.caseDedup[key] = obs
	log.Printf("auto-case created for rule=%s node=%s", ruleID, node)
}



func payloadJSON(env *erav1.Envelope) string {

	switch p := env.GetPayload().(type) {

	case *erav1.Envelope_Process:

		b, _ := json.Marshal(p.Process)

		return string(b)

	case *erav1.Envelope_Network:

		b, _ := json.Marshal(p.Network)

		return string(b)

	case *erav1.Envelope_Auth:

		b, _ := json.Marshal(p.Auth)

		return string(b)

	case *erav1.Envelope_File:

		b, _ := json.Marshal(p.File)

		return string(b)

	default:

		return ""

	}

}



func eventIDStr(b []byte) string {

	if len(b) == 16 {

		var id ulid.ULID

		copy(id[:], b)

		return id.String()

	}

	return uuid.NewString()

}



func parseNetwork(payload string) (srcIP, dstIP string, dstPort uint32) {

	var m map[string]any

	if err := json.Unmarshal([]byte(payload), &m); err != nil {

		return "", "", 0

	}

	srcIP, _ = m["src_ip"].(string)

	dstIP, _ = m["dst_ip"].(string)

	switch v := m["dst_port"].(type) {

	case float64:

		dstPort = uint32(v)

	}

	return srcIP, dstIP, dstPort

}


