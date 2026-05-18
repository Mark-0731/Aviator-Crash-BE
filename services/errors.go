package services

import "errors"

// Domain-specific errors for cashout flow
// These allow the websocket handler to provide user-friendly error messages
var (
	// ErrBetNotFound indicates the bet ID does not exist
	ErrBetNotFound = errors.New("bet not found")

	// ErrBetNotOwnedByUser indicates the bet belongs to a different user
	ErrBetNotOwnedByUser = errors.New("bet not owned by user")

	// ErrBetAlreadySettled indicates the bet has already been cashed out or lost
	ErrBetAlreadySettled = errors.New("bet already settled")

	// ErrRoundNotFound indicates the round ID does not exist
	ErrRoundNotFound = errors.New("round not found")

	// ErrRoundNotRunning indicates the round is not in running phase
	ErrRoundNotRunning = errors.New("round not running")

	// ErrCashoutTooLate indicates the current multiplier has reached or exceeded crash point
	ErrCashoutTooLate = errors.New("cashout too late - multiplier at or above crash point")

	// ErrEngineStateUnavailable indicates the game engine state is not accessible
	ErrEngineStateUnavailable = errors.New("game engine state unavailable")

	// ErrInsufficientBalance indicates user doesn't have enough balance
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrDuplicateBet indicates user already has a bet in this round
	ErrDuplicateBet = errors.New("duplicate bet in round")
)
