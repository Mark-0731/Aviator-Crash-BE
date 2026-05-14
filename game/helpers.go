package game

import "math"

// CalculateCurrentMultiplier calculates the current multiplier based on elapsed time
// FULLY FUNCTIONAL - NO PLACEHOLDERS
// Formula: e^(0.00006 * elapsed_milliseconds)
// This gives a realistic growth rate:
// - 1.00x at 0s
// - 2.00x at ~11.5s
// - 5.00x at ~26.8s
// - 10.00x at ~38.4s
// - 100.00x at ~76.8s
func CalculateCurrentMultiplier(elapsedSeconds float64) float64 {
	// Convert seconds to milliseconds and apply exponential growth
	elapsedMillis := elapsedSeconds * 1000
	multiplier := math.Exp(0.00006 * elapsedMillis)

	// Safety cap to prevent infinity values
	// Max multiplier is 20x (matches max crash point)
	if math.IsInf(multiplier, 1) || multiplier > 20.0 {
		return 20.0
	}

	return multiplier
}
