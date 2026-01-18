package pkgerrors

import (
	"errors"
	"fmt"
)

const (
	CodeNonExistingKey = -1001
	CodeJSONParsing    = -1002
	CodeDuplicateKey   = -1003
	CodeUnknown        = -9999
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

var (
	ErrDuplicateKey   = &AppError{Code: CodeDuplicateKey, Message: "duplicate key violation"}
	ErrNonExistingKey = &AppError{Code: CodeNonExistingKey, Message: "non-existing key"}
	ErrJSONParsing    = &AppError{Code: CodeJSONParsing, Message: "JSON parsing failed"}
)

func NewDuplicateKeyError(err error) *AppError {
	return &AppError{
		Code:    CodeDuplicateKey,
		Message: "duplicate key violation",
		Err:     err,
	}
}

func NewNonExistingKeyError(err error) *AppError {
	return &AppError{
		Code:    CodeNonExistingKey,
		Message: "key does not exist",
		Err:     err,
	}
}

func NewJSONParsingError(err error) *AppError {
	return &AppError{
		Code:    CodeJSONParsing,
		Message: "failed to parse JSON",
		Err:     err,
	}
}

func IsDuplicateKeyError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeDuplicateKey
	}
	return false
}

func IsNonExistingKeyError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeNonExistingKey
	}
	return false
}

func IsJSONParsingError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == CodeJSONParsing
	}
	return false
}

func GetErrorCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeUnknown
}
