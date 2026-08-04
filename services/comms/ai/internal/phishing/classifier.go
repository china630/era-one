package phishing

import (
	"strings"
)

type Message struct {
	From    string            `json:"from"`
	Subject string            `json:"subject"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

type Result struct {
	RiskScore int      `json:"risk_score"`
	Hints     []string `json:"hints"`
	Verdict   string   `json:"verdict"`
}

var urgencyTokens = []string{
	"urgent",
	"immediately",
	"verify your",
	"click here",
	"suspended",
	"password",
}

func Classify(msg Message) Result {
	score := 0
	var hints []string
	body := strings.ToLower(msg.Body)
	subject := strings.ToLower(msg.Subject)

	for _, token := range urgencyTokens {
		if strings.Contains(body, token) || strings.Contains(subject, token) {
			score += 15
			hints = append(hints, "urgency:"+token)
		}
	}

	fromDomain := extractDomain(msg.From)
	if replyTo := headerValue(msg.Headers, "reply-to"); replyTo != "" {
		replyDomain := extractDomain(replyTo)
		if fromDomain != "" && replyDomain != "" && fromDomain != replyDomain {
			score += 40
			hints = append(hints, "reply-to-mismatch")
		}
	}

	if strings.Contains(msg.From, "<") && strings.Contains(strings.ToLower(msg.From), "ceo") {
		if fromDomain != "" && !strings.HasSuffix(fromDomain, ".local") && !strings.HasSuffix(fromDomain, ".gov.az") {
			score += 25
			hints = append(hints, "executive-spoof-domain")
		}
	}

	if score > 100 {
		score = 100
	}

	verdict := "benign"
	switch {
	case score >= 60:
		verdict = "malicious"
	case score >= 30:
		verdict = "suspicious"
	}

	return Result{RiskScore: score, Hints: hints, Verdict: verdict}
}

func headerValue(headers map[string]string, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

func extractDomain(addr string) string {
	addr = strings.TrimSpace(addr)
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		domain := addr[i+1:]
		if j := strings.IndexAny(domain, "> "); j >= 0 {
			domain = domain[:j]
		}
		return strings.ToLower(strings.Trim(domain, ">"))
	}
	return ""
}
