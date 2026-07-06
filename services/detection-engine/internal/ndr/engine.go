// Package ndr — Network Detection & Response (T1021, C2 beaconing, DNS tunnel).
package ndr

import (
	"encoding/json"
	"math"
	"strings"
	"sync"
	"time"
)

type FlowRecord struct {
	SrcIP    string
	DstIP    string
	DstPort  uint32
	Observed time.Time
}

type dnsRecord struct {
	Query    string
	Observed time.Time
}

type Engine struct {
	mu     sync.Mutex
	window time.Duration
	bySrc  map[string][]FlowRecord
	beacon map[string][]time.Time
	dns    map[string][]dnsRecord
	exfil  map[string][]exfilRecord
	ja3    map[string]string
}

type exfilRecord struct {
	Bytes    uint64
	JA3      string
	Observed time.Time
}

// Известные вредоносные JA3 (синтетика для golden/eval).
var suspiciousJA3 = map[string]string{
	"e7d705a3286e19ea42f587b344ee6865": "era-ndr-ja3-cobalt-strike",
	"6734fbd316a9b2c9dc6fd41a41d7d051": "era-ndr-ja3-trickbot",
}

func New(window time.Duration) *Engine {
	return &Engine{
		window: window,
		bySrc:  make(map[string][]FlowRecord),
		beacon: make(map[string][]time.Time),
		dns:    make(map[string][]dnsRecord),
		exfil:  make(map[string][]exfilRecord),
		ja3:    make(map[string]string),
	}
}

func (e *Engine) Observe(srcIP, dstIP string, dstPort uint32, at time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	cutoff := at.Add(-e.window)
	list := e.bySrc[srcIP]
	var kept []FlowRecord
	for _, r := range list {
		if r.Observed.After(cutoff) {
			kept = append(kept, r)
		}
	}
	kept = append(kept, FlowRecord{SrcIP: srcIP, DstIP: dstIP, DstPort: dstPort, Observed: at})
	e.bySrc[srcIP] = kept
}

// LateralMovement T1021: один источник -> несколько хостов на SMB/RDP/WinRM.
func (e *Engine) LateralMovement(srcIP string) (bool, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	list := e.bySrc[srcIP]
	targets := map[string]bool{}
	lateralPorts := map[uint32]bool{445: true, 3389: true, 5985: true}
	for _, r := range list {
		if lateralPorts[r.DstPort] {
			targets[r.DstIP] = true
		}
	}
	if len(targets) >= 2 {
		return true, "era-ndr-t1021-lateral-movement"
	}
	return false, ""
}

func IsLateralPort(port uint32) bool {
	return port == 445 || port == 3389 || port == 5985
}

func PayloadHasLateral(payload string) bool {
	low := strings.ToLower(payload)
	for _, p := range []string{`"dst_port":445`, `"dst_port":3389`, `"dst_port":5985`} {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

// ObserveBeacon фиксирует исходящие соединения к одному dst для эвристики C2 beaconing.
func (e *Engine) ObserveBeacon(srcIP, dstIP string, at time.Time) {
	if srcIP == "" || dstIP == "" {
		return
	}
	key := srcIP + "->" + dstIP
	e.mu.Lock()
	defer e.mu.Unlock()
	cutoff := at.Add(-e.window)
	list := e.beacon[key]
	var kept []time.Time
	for _, ts := range list {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	kept = append(kept, at)
	e.beacon[key] = kept
}

// Beaconing T1071: ≥5 соединений с низкой дисперсией интервалов (регулярный C2).
func (e *Engine) Beaconing(srcIP, dstIP string) (bool, string) {
	key := srcIP + "->" + dstIP
	e.mu.Lock()
	defer e.mu.Unlock()
	times := e.beacon[key]
	if len(times) < 5 {
		return false, ""
	}
	var intervals []float64
	for i := 1; i < len(times); i++ {
		intervals = append(intervals, times[i].Sub(times[i-1]).Seconds())
	}
	mean := 0.0
	for _, v := range intervals {
		mean += v
	}
	mean /= float64(len(intervals))
	var variance float64
	for _, v := range intervals {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(intervals))
	if mean > 0 && math.Sqrt(variance)/mean < 0.25 {
		return true, "era-ndr-t1071-beaconing"
	}
	return false, ""
}

// ObserveDNS фиксирует DNS-запросы источника.
func (e *Engine) ObserveDNS(srcIP, query string, at time.Time) {
	if srcIP == "" || query == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	cutoff := at.Add(-e.window)
	list := e.dns[srcIP]
	var kept []dnsRecord
	for _, r := range list {
		if r.Observed.After(cutoff) {
			kept = append(kept, r)
		}
	}
	kept = append(kept, dnsRecord{Query: query, Observed: at})
	e.dns[srcIP] = kept
}

// DNSTunnel T1071.004: длинные/высокоэнтропийные поддомены или много уникальных запросов.
func (e *Engine) DNSTunnel(srcIP string) (bool, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	list := e.dns[srcIP]
	if len(list) < 8 {
		return false, ""
	}
	unique := map[string]bool{}
	longLabels := 0
	for _, r := range list {
		unique[r.Query] = true
		if longestLabel(r.Query) >= 48 || shannonEntropy(r.Query) >= 3.8 {
			longLabels++
		}
	}
	if longLabels >= 3 || len(unique) >= 12 {
		return true, "era-ndr-t1071-004-dns-tunnel"
	}
	return false, ""
}

// ObserveExfil фиксирует исходящий объём и JA3 fingerprint.
func (e *Engine) ObserveExfil(srcIP string, bytes uint64, ja3 string, at time.Time) {
	if srcIP == "" || bytes == 0 {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	cutoff := at.Add(-e.window)
	list := e.exfil[srcIP]
	var kept []exfilRecord
	for _, r := range list {
		if r.Observed.After(cutoff) {
			kept = append(kept, r)
		}
	}
	kept = append(kept, exfilRecord{Bytes: bytes, JA3: ja3, Observed: at})
	e.exfil[srcIP] = kept
	if ja3 != "" {
		e.ja3[srcIP] = ja3
	}
}

// DataExfil T1048: суммарный исходящий объём превышает порог в окне.
func (e *Engine) DataExfil(srcIP string) (bool, string) {
	const threshold = uint64(50 * 1024 * 1024) // 50 MiB
	e.mu.Lock()
	defer e.mu.Unlock()
	var total uint64
	for _, r := range e.exfil[srcIP] {
		total += r.Bytes
	}
	if total >= threshold {
		return true, "era-ndr-t1048-data-exfil"
	}
	return false, ""
}

// JA3Fingerprint T1071: совпадение с известным вредоносным JA3.
func (e *Engine) JA3Fingerprint(srcIP string) (bool, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	ja3 := e.ja3[srcIP]
	if ja3 == "" {
		return false, ""
	}
	if rule, ok := suspiciousJA3[strings.ToLower(ja3)]; ok {
		return true, rule
	}
	return false, ""
}

// ParseNetworkExfil извлекает bytes и ja3 из JSON network payload.
func ParseNetworkExfil(payload string) (bytes uint64, ja3 string) {
	var m map[string]any
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return 0, ""
	}
	switch v := m["bytes_out"].(type) {
	case float64:
		bytes = uint64(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			bytes = uint64(n)
		}
	}
	for _, k := range []string{"ja3", "ja3_fingerprint", "tls_ja3"} {
		if v, ok := m[k].(string); ok && v != "" {
			return bytes, v
		}
	}
	return bytes, ""
}

func longestLabel(q string) int {
	q = strings.TrimSuffix(q, ".")
	parts := strings.Split(q, ".")
	max := 0
	for _, p := range parts {
		if len(p) > max {
			max = len(p)
		}
	}
	return max
}

func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	freq := map[rune]int{}
	for _, r := range s {
		freq[r]++
	}
	var entropy float64
	n := float64(len(s))
	for _, c := range freq {
		p := float64(c) / n
		entropy -= p * math.Log2(p)
	}
	return entropy
}

// ParseDNS извлекает src и query из JSON dns/network payload.
func ParseDNS(payload string) (srcIP, query string) {
	var m map[string]any
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return "", ""
	}
	srcIP, _ = m["src_ip"].(string)
	for _, k := range []string{"query", "dns_query", "qname"} {
		if v, ok := m[k].(string); ok && v != "" {
			return srcIP, v
		}
	}
	return srcIP, ""
}
