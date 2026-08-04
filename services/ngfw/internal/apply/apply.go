// Package apply — optional host firewall applicator (Linux lab, Phase 2).
package apply

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"era/services/ngfw/internal/policy"
)

// Backend applies deny rules to the host.
type Backend interface {
	ApplyDeny(rule policy.Rule) error
	Name() string
}

// Enabled reports ERA_NGFW_APPLY=1.
func Enabled() bool {
	return os.Getenv("ERA_NGFW_APPLY") == "1"
}

// Select picks nftables/iptables on Linux when enabled; otherwise noop.
func Select() Backend {
	if !Enabled() || runtime.GOOS != "linux" {
		return Noop{}
	}
	if _, err := exec.LookPath("nft"); err == nil {
		return NFT{}
	}
	if _, err := exec.LookPath("iptables"); err == nil {
		return IPTables{}
	}
	return Noop{}
}

type Noop struct{}

func (Noop) Name() string { return "noop" }
func (Noop) ApplyDeny(policy.Rule) error {
	return nil
}

type NFT struct{}

func (NFT) Name() string { return "nftables" }
func (NFT) ApplyDeny(r policy.Rule) error {
	// Lab-only: add inet filter drop for dst port (best-effort, may need root).
	if r.DstPort == 0 {
		return fmt.Errorf("dst_port required for apply")
	}
	cmd := exec.Command("nft", "add", "rule", "inet", "filter", "input", "tcp", "dport",
		fmt.Sprintf("%d", r.DstPort), "drop")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nft: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

type IPTables struct{}

func (IPTables) Name() string { return "iptables" }
func (IPTables) ApplyDeny(r policy.Rule) error {
	if r.DstPort == 0 {
		return fmt.Errorf("dst_port required")
	}
	cmd := exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport",
		fmt.Sprintf("%d", r.DstPort), "-j", "DROP")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// DryRun formats what would be applied (tests / Windows).
func DryRun(r policy.Rule) string {
	return fmt.Sprintf("deny tcp/%d src=%s dst=%s", r.DstPort, r.SrcCIDR, r.DstCIDR)
}
