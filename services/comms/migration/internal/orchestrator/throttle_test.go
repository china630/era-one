package orchestrator

import (
	"testing"
	"time"
)

func TestHostThrottleAllowsBurst(t *testing.T) {
	th := NewHostThrottle(2, 200*time.Millisecond)
	start := time.Now()
	th.Allow("cg.lab.local")
	th.Allow("cg.lab.local")
	th.Allow("cg.lab.local")
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("expected throttle delay on third call")
	}
}
