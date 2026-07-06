// Package dp — differential privacy для агрегатов угроз (F4-3/F4-4).
package dp

import "math"

// NoisyCount добавляет Laplace-шум к счётчику публикаций (air-gap aggregate).
func NoisyCount(count int, epsilon float64) float64 {
	if epsilon <= 0 {
		epsilon = 1.0
	}
	scale := 1.0 / epsilon
	u := randUniform()
	sign := 1.0
	if u < 0 {
		sign = -1
	}
	return float64(count) + (-scale * sign * math.Log(1-2*math.Abs(u)))
}

func randUniform() float64 {
	// deterministic enough for tests via seed in test file
	return pseudoRand()
}

var seed uint64 = 42

func pseudoRand() float64 {
	seed = seed*6364136223846793005 + 1
	return float64(int64(seed>>33)%1000)/500.0 - 1.0
}

func ResetTestSeed() { seed = 42 }
