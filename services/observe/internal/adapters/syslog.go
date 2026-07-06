package adapters

import (
	"errors"
	"strings"
)

// ParseSyslogNetwork — упрощённый syslog (PRTG/Zabbix trap) → network alert fields.
func ParseSyslogNetwork(line string) (host, summary string, err error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", errors.New("empty syslog")
	}
	// формат: host|message или RFC5424 tail
	if strings.Contains(line, "|") {
		parts := strings.SplitN(line, "|", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", line, nil
	}
	host = fields[len(fields)-2]
	summary = fields[len(fields)-1]
	return host, summary, nil
}
