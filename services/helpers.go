package services

import (
	"aviator-backend/models"
)

// convertToUserResponses converts a slice of users to user responses
func convertToUserResponses(users []models.User) []models.UserResponse {
	responses := make([]models.UserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}
	return responses
}

// convertToRoundResponses converts a slice of rounds to round responses
func convertToRoundResponses(rounds []models.Round) []models.RoundResponse {
	responses := make([]models.RoundResponse, len(rounds))
	for i, round := range rounds {
		responses[i] = round.ToResponse()
	}
	return responses
}

// convertToBetResponses converts a slice of bets to bet responses
func convertToBetResponses(bets []models.Bet) []models.BetResponse {
	responses := make([]models.BetResponse, len(bets))
	for i, bet := range bets {
		responses[i] = bet.ToResponse()
	}
	return responses
}

// convertToTransactionResponses converts a slice of transactions to transaction responses
func convertToTransactionResponses(transactions []models.Transaction) []models.TransactionResponse {
	responses := make([]models.TransactionResponse, len(transactions))
	for i, tx := range transactions {
		responses[i] = tx.ToResponse()
	}
	return responses
}

// abs returns absolute value of int64
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
