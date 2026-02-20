package errors

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeValidationError  ErrorCode = "VALIDATION_ERROR"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeInternal         ErrorCode = "INTERNAL_ERROR"
	CodeNotSupported     ErrorCode = "NOT_SUPPORTED"
	CodePermissionDenied ErrorCode = "PERMISSION_DENIED"
)

type DomainError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func New(code ErrorCode, msg string) error {
	return &DomainError{Code: code, Message: msg}
}

func Wrap(err error, code ErrorCode, msg string) error {
	return &DomainError{Code: code, Message: msg, Err: err}
}

// IsCode checks if an error has a specific error code.
func IsCode(err error, code ErrorCode) bool {
	var de *DomainError
	if errors.As(err, &de) {
		return de.Code == code
	}
	return false
}
