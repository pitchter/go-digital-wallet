package apperror

import "net/http"

// AppError is the canonical application error returned by services.
type AppError struct {
	StatusCode int
	Code       string
	Message    string
	Details    any
	Err        error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

// Unwrap returns the underlying error, if present.
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// WithDetails returns a copy with details attached.
func (e *AppError) WithDetails(details any) *AppError {
	if e == nil {
		return nil
	}

	out := *e
	out.Details = details
	return &out
}

// WithErr returns a copy with a wrapped error attached.
func (e *AppError) WithErr(err error) *AppError {
	if e == nil {
		return nil
	}

	out := *e
	out.Err = err
	return &out
}

// As converts an error into an AppError when possible.
func As(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}

	appErr, ok := err.(*AppError)
	return appErr, ok
}

// BadRequest returns a malformed-request error.
func BadRequest(message string) *AppError {
	return &AppError{
		StatusCode: http.StatusBadRequest,
		Code:       "bad_request",
		Message:    message,
	}
}

// Validation returns a validation error.
func Validation(message string) *AppError {
	return &AppError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       "validation_error",
		Message:    message,
	}
}

// NotFound returns a missing-resource error.
func NotFound(message string) *AppError {
	return &AppError{
		StatusCode: http.StatusNotFound,
		Code:       "resource_not_found",
		Message:    message,
	}
}

// Conflict returns a resource conflict error.
func Conflict(message string) *AppError {
	return &AppError{
		StatusCode: http.StatusConflict,
		Code:       "resource_conflict",
		Message:    message,
	}
}

// IdempotencyConflict returns an idempotency key conflict error.
func IdempotencyConflict() *AppError {
	return &AppError{
		StatusCode: http.StatusConflict,
		Code:       "idempotency_conflict",
		Message:    "idempotency key was already used for a different request",
	}
}

// InsufficientFunds returns a wallet balance error.
func InsufficientFunds() *AppError {
	return &AppError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       "insufficient_funds",
		Message:    "wallet balance is insufficient",
	}
}

// WalletNotActive returns an inactive wallet error.
func WalletNotActive() *AppError {
	return &AppError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       "wallet_not_active",
		Message:    "wallet is not active",
	}
}

// AlreadyReversed returns an already-reversed error.
func AlreadyReversed() *AppError {
	return &AppError{
		StatusCode: http.StatusConflict,
		Code:       "already_reversed",
		Message:    "transaction has already been reversed",
	}
}

// Internal returns a generic internal error.
func Internal(err error) *AppError {
	return &AppError{
		StatusCode: http.StatusInternalServerError,
		Code:       "internal_error",
		Message:    "internal server error",
		Err:        err,
	}
}
