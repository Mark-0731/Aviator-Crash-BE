package models

import "fmt"

// FormatCents converts cents to dollar string
func FormatCents(cents int64) string {
	dollars := float64(cents) / 100.0
	if cents >= 0 {
		return fmt.Sprintf("$%.2f", dollars)
	}
	return fmt.Sprintf("-$%.2f", -dollars)
}

// convertX100ToFloat converts X100 integer to float64
func convertX100ToFloat(x100 int64) float64 {
	return float64(x100) / 100.0
}

// calculateHouseProfit calculates house profit from wagered and payout
func calculateHouseProfit(wageredCents, payoutCents int64) int64 {
	return wageredCents - payoutCents
}
