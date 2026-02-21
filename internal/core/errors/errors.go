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
	Context map[string]interface{}
}

const (
	CtxPath      = "path"
	CtxOperation = "operation"
	CtxLanguage  = "language"
	CtxSymbol    = "symbol"
)

func (e *DomainError) WithContext(key string, value interface{}) *DomainError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

func (e *DomainError) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Code, e.Message)
	if e.Err != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Err)
	}
	if len(e.Context) > 0 {
		msg += fmt.Sprintf(" %v", e.Context)
	}
	return msg
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
func AddContext(err error, key string, value interface{}) error {
	var de *DomainError
	if errors.As(err, &de) {
		de.WithContext(key, value)
		return de
	}
	return &DomainError{
		Code:    CodeInternal,
		Message: "wrapped error",
		Err:     err,
		Context: map[string]interface{}{key: value},
	}
}

func IsCode(err error, code ErrorCode) bool {
	var de *DomainError
	if errors.As(err, &de) {
		return de.Code == code
	}
	return false
}
