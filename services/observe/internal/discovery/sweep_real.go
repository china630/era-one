package discovery

import (
	"context"
	"net"
	"os"
	"strings"
	"time"
)

// Sweep выбирает sim или real; в ERA_PRODUCTION без ERA_DISCOVERY_SIM — без fallback на sim.
func Sweep(cidr string) []Node {
	if os.Getenv("ERA_DISCOVERY_SIM") == "1" {
		return SweepSimulated(cidr)
	}
	nodes, err := SweepReal(cidr)
	if err != nil || len(nodes) == 0 {
		if os.Getenv("ERA_PRODUCTION") == "1" || os.Getenv("ERA_OBSERVE_STRICT") == "1" {
			return nil
		}
		return SweepSimulated(cidr)
	}
	return nodes
}

// SweepReal — ping sweep с rate-limit и ERA_DISCOVERY_ALLOWLIST.
func SweepReal(cidr string) ([]Node, error) {
	if cidr == "" {
		cidr = "127.0.0.1/32"
	}
	ips, err := hostsInAllowlist(cidr)
	if err != nil {
		return nil, err
	}
	var out []Node
	for _, ip := range ips {
		if pingHost(ip, 300*time.Millisecond) {
			out = append(out, Node{IP: ip, Kind: "unknown"})
		}
		time.Sleep(10 * time.Millisecond)
	}
	return out, nil
}

func hostsInAllowlist(cidr string) ([]string, error) {
	allow := parseAllowlist(os.Getenv("ERA_DISCOVERY_ALLOWLIST"))
	if len(allow) == 0 {
		allow = []string{cidr}
	}
	var ips []string
	seen := map[string]struct{}{}
	for _, block := range allow {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		_, network, err := net.ParseCIDR(block)
		if err != nil {
			if ip := net.ParseIP(block); ip != nil {
				if _, ok := seen[block]; !ok {
					seen[block] = struct{}{}
					ips = append(ips, block)
				}
			}
			continue
		}
		for ip := network.IP.Mask(network.Mask); network.Contains(ip); incIP(ip) {
			s := ip.String()
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			ips = append(ips, s)
			if len(ips) >= 64 {
				return ips, nil
			}
		}
	}
	return ips, nil
}

func parseAllowlist(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func pingHost(ip string, timeout time.Duration) bool {
	ports := probePorts()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var d net.Dialer
	for _, port := range ports {
		conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
		if err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}

func probePorts() []string {
	if raw := os.Getenv("ERA_DISCOVERY_PROBE_PORTS"); raw != "" {
		var ports []string
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				ports = append(ports, p)
			}
		}
		if len(ports) > 0 {
			return ports
		}
	}
	return []string{"22", "80"}
}
