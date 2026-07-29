// Package notify — уведомления и signed action links (AC-MM-2,7).
package notify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Mailer — доставка служебных писем (тесты: Recorder).
type Mailer interface {
	Send(from string, to []string, subject, body string) error
}

// Recorder — in-memory mailer.
type Recorder struct {
	mu   sync.Mutex
	Sent []Mail
}

type Mail struct {
	From, Subject, Body string
	To                  []string
}

func (r *Recorder) Send(from string, to []string, subject, body string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Sent = append(r.Sent, Mail{From: from, To: append([]string(nil), to...), Subject: subject, Body: body})
	return nil
}

// Tokens — HMAC one-time action tokens.
type Tokens struct {
	Secret []byte
	TTL    time.Duration
	now    func() time.Time
	mu     sync.Mutex
	used   map[string]struct{}
}

func NewTokens(secret []byte) *Tokens {
	if len(secret) == 0 {
		secret = []byte("dev-only-mm-secret-change-me")
	}
	return &Tokens{
		Secret: secret,
		TTL:    24 * time.Hour,
		now:    time.Now,
		used:   make(map[string]struct{}),
	}
}

type claims struct {
	HoldID    string `json:"h"`
	Moderator string `json:"m"`
	Action    string `json:"a"` // approve|reject
	Exp       int64  `json:"e"`
}

// Issue создаёт token для approve/reject.
func (t *Tokens) Issue(holdID, moderator, action string) (string, error) {
	action = strings.ToLower(action)
	if action != "approve" && action != "reject" {
		return "", fmt.Errorf("bad action")
	}
	c := claims{
		HoldID:    holdID,
		Moderator: moderator,
		Action:    action,
		Exp:       t.now().Add(t.TTL).Unix(),
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, t.Secret)
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig, nil
}

// Consume проверяет token и помечает one-time.
func (t *Tokens) Consume(token string) (holdID, moderator, action string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("malformed token")
	}
	mac := hmac.New(sha256.New, t.Secret)
	_, _ = mac.Write([]byte(parts[0]))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[1])) {
		return "", "", "", fmt.Errorf("bad signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", "", err
	}
	var c claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return "", "", "", err
	}
	if t.now().Unix() > c.Exp {
		return "", "", "", fmt.Errorf("token expired")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.used[token]; ok {
		return "", "", "", fmt.Errorf("token already used")
	}
	t.used[token] = struct{}{}
	return c.HoldID, c.Moderator, c.Action, nil
}

// Service — шаблоны писем модератору/отправителю.
type Service struct {
	From       string
	PublicBase string // https://mm.lab.local
	Mailer     Mailer
	Tokens     *Tokens
}

func (s *Service) NotifyModerator(holdID, sender, subject string, moderators []string, preview string) error {
	var links []string
	for _, m := range moderators {
		ap, err := s.Tokens.Issue(holdID, m, "approve")
		if err != nil {
			return err
		}
		rj, err := s.Tokens.Issue(holdID, m, "reject")
		if err != nil {
			return err
		}
		base := strings.TrimRight(s.PublicBase, "/")
		links = append(links, fmt.Sprintf("Moderator %s:\n  Approve: %s/v1/moderation/action?token=%s\n  Reject:  %s/v1/moderation/action?token=%s&comment=REASON\n",
			m, base, ap, base, rj))
	}
	body := fmt.Sprintf("[ERA Mail Moderation]\nSender: %s\nSubject: %s\nHold: %s\n\n%s\n\n--- preview ---\n%s\n",
		sender, subject, holdID, strings.Join(links, "\n"), preview)
	from := s.From
	if from == "" {
		from = "moderation@localhost"
	}
	return s.Mailer.Send(from, moderators, "[Moderation] "+subject, body)
}

func (s *Service) NotifySenderHeld(sender, subject, holdID string) error {
	from := s.From
	if from == "" {
		from = "moderation@localhost"
	}
	body := fmt.Sprintf("Your message %q is pending manager approval (hold %s).\n", subject, holdID)
	return s.Mailer.Send(from, []string{sender}, "[Pending moderation] "+subject, body)
}

func (s *Service) NotifySenderRejected(sender, subject, comment string) error {
	from := s.From
	if from == "" {
		from = "moderation@localhost"
	}
	body := fmt.Sprintf("Your message %q was rejected.\nComment: %s\n", subject, comment)
	return s.Mailer.Send(from, []string{sender}, "[Rejected] "+subject, body)
}
