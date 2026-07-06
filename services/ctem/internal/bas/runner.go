// Package bas — CTEM/BAS симуляция attack chain (ADR-0006 P1, Фаза 3).
package bas

import (
	"context"

	"era/services/platform/envelope"
)

type Runner struct {
	Pub *envelope.Publisher
}

func (r *Runner) SimulateLateral(ctx context.Context, srcIP string) error {
	if r == nil || r.Pub == nil {
		return nil
	}
	targets := []struct {
		dst  string
		port uint32
	}{
		{"10.0.0.10", 445},
		{"10.0.0.11", 3389},
		{"10.0.0.12", 5985},
	}
	for _, t := range targets {
		if err := r.Pub.PublishNetwork(ctx, srcIP, t.dst, "tcp", "outbound", t.port); err != nil {
			return err
		}
	}
	return nil
}
