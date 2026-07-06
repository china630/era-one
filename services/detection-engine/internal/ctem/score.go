// Package ctem — continuous threat exposure score (ADR-0006 G-03).
package ctem

// Score связывает VM findings и exposure в единый risk 0..100.
func Score(vmFindings int, exposureWeight float64) float64 {
	if vmFindings < 0 {
		vmFindings = 0
	}
	if exposureWeight < 0 {
		exposureWeight = 0
	}
	if exposureWeight > 1 {
		exposureWeight = 1
	}
	base := float64(vmFindings) * 8.0
	return base + exposureWeight*40.0
}
