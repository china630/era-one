package dp

import "testing"

func TestNoisyCount(t *testing.T) {
	ResetTestSeed()
	n := NoisyCount(100, 2.0)
	if n < 90 || n > 110 {
		t.Fatalf("unexpected noisy count: %v", n)
	}
}
