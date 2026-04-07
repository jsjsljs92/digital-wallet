package dbmodel

import "fmt"

type ErrorCode string

const (
	ErrInsufficientBalance  ErrorCode = "INSUFFICIENT_BALANCE"
	ErrDailyLimitExceeded   ErrorCode = "DAILY_LIMIT_EXCEEDED"
	ErrWeeklyLimitExceeded  ErrorCode = "WEEKLY_LIMIT_EXCEEDED"
	ErrWalletNotFound       ErrorCode = "WALLET_NOT_FOUND"
	ErrWalletAlreadyExists  ErrorCode = "WALLET_ALREADY_EXISTS"
	ErrInvalidAmount        ErrorCode = "INVALID_AMOUNT"
	ErrTransactionNotFound  ErrorCode = "TRANSACTION_NOT_FOUND"
	ErrOptimisticLockFailed ErrorCode = "OPTIMISTIC_LOCK_FAILED"
	ErrIdempotencyConflict  ErrorCode = "IDEMPOTENCY_CONFLICT"
	ErrInvalidInput         ErrorCode = "INVALID_INPUT"
	ErrRateLimitExceeded    ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrInternalServer       ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrUnauthorized         ErrorCode = "UNAUTHORIZED"
	ErrInvalidOTP           ErrorCode = "INVALID_OTP"
)

type DomainError struct {
	Code    ErrorCode
	Message string
	Details map[string]interface{}
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewDomainError(code ErrorCode, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

func (e *DomainError) WithDetail(key string, value interface{}) *DomainError {
	e.Details[key] = value
	return e
}
