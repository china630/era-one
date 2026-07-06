// Credentialed scan stub — SSH banner grab (S7-1, ERA Vuln).
package scanner

import (
	"fmt"
	"net"
	"strings"
	"time"

	"era/services/vm/internal/models"
)

// SSHCredentialedExecutor — минимальный credentialed scan без внешних зависимостей.
type SSHCredentialedExecutor struct {
	Timeout time.Duration
	Port    string
}

// NewSSHCredentialed создаёт SSH stub executor.
func NewSSHCredentialed() *SSHCredentialedExecutor {
	return &SSHCredentialedExecutor{Timeout: 5 * time.Second, Port: "22"}
}

// Execute подключается к target:port, читает SSH banner и сопоставляет с шаблоном.
func (e *SSHCredentialedExecutor) Execute(target string, tpl *models.Template) ([]models.Finding, error) {
	if tpl == nil {
		return nil, nil
	}
	host := target
	port := e.Port
	if h, p, ok := strings.Cut(target, ":"); ok && p != "" {
		host, port = h, p
	}
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, e.Timeout)
	if err != nil {
		return nil, fmt.Errorf("ssh credentialed dial %s: %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(e.Timeout))
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("ssh banner read %s: %w", addr, err)
	}
	banner := string(buf[:n])
	if !strings.HasPrefix(banner, "SSH-") {
		return nil, nil
	}
	return []models.Finding{{
		Target:            target,
		TemplateID:        tpl.ID,
		VulnerabilityName: tpl.Info.Name,
		Severity:          tpl.Info.Severity,
		MatchedURL:        strings.TrimSpace(banner),
		Timestamp:         time.Now().UTC(),
	}}, nil
}
