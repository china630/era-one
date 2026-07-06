// Package netflow — базовый parse flow records (MVP line format).
package netflow

import (
	"errors"
	"strconv"
	"strings"
)

type Record struct {
	SrcIP   string `json:"src_ip"`
	DstIP   string `json:"dst_ip"`
	SrcPort uint32 `json:"src_port"`
	DstPort uint32 `json:"dst_port"`
	Proto   string `json:"proto"`
	Bytes   uint64 `json:"bytes"`
}

// ParseLine — CSV: src,dst,srcport,dstport,proto,bytes
func ParseLine(line string) (Record, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return Record{}, errors.New("empty")
	}
	parts := strings.Split(line, ",")
	if len(parts) < 6 {
		return Record{}, errors.New("need 6 fields")
	}
	sp, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 32)
	dp, _ := strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 32)
	b, _ := strconv.ParseUint(strings.TrimSpace(parts[5]), 10, 64)
	return Record{
		SrcIP: strings.TrimSpace(parts[0]),
		DstIP: strings.TrimSpace(parts[1]),
		SrcPort: uint32(sp), DstPort: uint32(dp),
		Proto: strings.TrimSpace(parts[4]),
		Bytes: b,
	}, nil
}
