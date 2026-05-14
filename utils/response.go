package utils

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Common errors
var (
	ErrDuplicateBetError = errors.New("duplicate bet")
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	ErrInvalidPhase        ErrorCode = "INVALID_PHASE"
	ErrInsufficientBalance ErrorCode = "INSUFFICIENT_BALANCE"
	ErrDuplicateBet        ErrorCode = "DUPLICATE_BET"
	ErrAlreadyCashedOut    ErrorCode = "ALREADY_CASHED_OUT"
	ErrInvalidAmount       ErrorCode = "INVALID_AMOUNT"
	ErrAccountBanned       ErrorCode = "ACCOUNT_BANNED"
	ErrRateLimited         ErrorCode = "RATE_LIMITED"
	ErrUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrTicketExpired       ErrorCode = "TICKET_EXPIRED"
	ErrValidationError     ErrorCode = "VALIDATION_ERROR"
	ErrMaxWinExceeded      ErrorCode = "MAX_WIN_EXCEEDED"
	ErrInternalError       ErrorCode = "INTERNAL_ERROR"
	ErrNotFound            ErrorCode = "NOT_FOUND"
	ErrForbidden           ErrorCode = "FORBIDDEN"
	ErrBadRequest          ErrorCode = "BAD_REQUEST"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Success bool      `json:"success"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// RespondError sends a standardized error response
func RespondError(c *gin.Context, statusCode int, code ErrorCode, message string) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Code:    code,
		Message: message,
	})
}

// RespondSuccess sends a standardized success response
func RespondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// RespondSuccessWithMessage sends a success response with a message
func RespondSuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// RespondCreated sends a 201 Created response
func RespondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// GetHTTPStatusForError returns appropriate HTTP status code for error code
func GetHTTPStatusForError(code ErrorCode) int {
	switch code {
	case ErrUnauthorized, ErrTicketExpired:
		return http.StatusUnauthorized
	case ErrForbidden, ErrAccountBanned:
		return http.StatusForbidden
	case ErrNotFound:
		return http.StatusNotFound
	case ErrRateLimited:
		return http.StatusTooManyRequests
	case ErrInvalidPhase, ErrInsufficientBalance, ErrDuplicateBet,
		ErrAlreadyCashedOut, ErrInvalidAmount, ErrValidationError, ErrBadRequest:
		return http.StatusBadRequest
	case ErrInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

// RespondWithError is a convenience function that determines status code automatically
func RespondWithError(c *gin.Context, code ErrorCode, message string) {
	statusCode := GetHTTPStatusForError(code)
	RespondError(c, statusCode, code, message)
}
