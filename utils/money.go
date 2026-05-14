package utils

import (
	"fmt"
	"math"

	"aviator-backend/config"
)

// ToCents converts a dollar amount to cents
// CRITICAL: All monetary values stored as int64 cents to eliminate floating point drift
func ToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

// FromCents converts cents to dollar amount
func FromCents(cents int64) float64 {
	return float64(cents) / 100.0
}

// FormatCents formats cents as a dollar string
func FormatCents(cents int64) string {
	dollars := FromCents(cents)
	if cents >= 0 {
		return fmt.Sprintf("$%.2f", dollars)
	}
	return fmt.Sprintf("-$%.2f", math.Abs(dollars))
}

// ValidateAmount validates a bet amount in cents
func ValidateAmount(cents int64) error {
	if cents < config.AppConfig.MinBetCents {
		return fmt.Errorf("minimum bet is %s", FormatCents(config.AppConfig.MinBetCents))
	}
	if cents > config.AppConfig.MaxBetCents {
		return fmt.Errorf("maximum bet is %s", FormatCents(config.AppConfig.MaxBetCents))
	}
	return nil
}

// CalculatePayout calculates payout using integer arithmetic only
// CRITICAL: No float multiplication - prevents floating point errors
func CalculatePayout(betCents int64, multiplierX100 int64) int64 {
	return (betCents * multiplierX100) / 100
}

// CalculateProfit calculates profit (payout - bet amount)
func CalculateProfit(betCents int64, payoutCents int64) int64 {
	return payoutCents - betCents
}

// ApplyMaxWinCap applies the maximum win cap if necessary
func ApplyMaxWinCap(payoutCents int64) (cappedPayout int64, wasCapped bool) {
	if payoutCents > config.AppConfig.MaxWinCents {
		return config.AppConfig.MaxWinCents, true
	}
	return payoutCents, false
}

// MultiplierToX100 converts a float multiplier to int64 x100 format
// Example: 2.45 -> 245
func MultiplierToX100(multiplier float64) int64 {
	return int64(math.Floor(multiplier * 100))
}

// MultiplierFromX100 converts int64 x100 format to float multiplier
// Example: 245 -> 2.45
func MultiplierFromX100(multiplierX100 int64) float64 {
	return float64(multiplierX100) / 100.0
}
