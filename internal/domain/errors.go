package domain

import "fmt"

type ErrorCode string

const (
	CodeInvalid         ErrorCode = "INVALID_ARGUMENT"
	CodeNotFound        ErrorCode = "NOT_FOUND"
	CodeConflict        ErrorCode = "CONFLICT"
	CodeVersionConflict ErrorCode = "VERSION_CONFLICT"
	CodeIdempotency     ErrorCode = "IDEMPOTENCY_CONFLICT"
	CodeState           ErrorCode = "INVALID_STATE"
	CodeEvidenceMissing ErrorCode = "EVIDENCE_INCOMPLETE"
	CodeFrozen          ErrorCode = "BATCH_FROZEN"
	CodeForbidden       ErrorCode = "FORBIDDEN"
)

type BusinessError struct {
	Code    ErrorCode
	Message string
	Fields  map[string]string
}

func (e *BusinessError) Error() string { return e.Message }

func NewError(code ErrorCode, message string) error {
	return &BusinessError{Code: code, Message: message}
}

func FieldError(field, message string) error {
	return &BusinessError{Code: CodeInvalid, Message: fmt.Sprintf("%s：%s", field, message), Fields: map[string]string{field: message}}
}

func ErrorCodeOf(err error) ErrorCode {
	if typed, ok := err.(*BusinessError); ok {
		return typed.Code
	}
	return "INTERNAL_ERROR"
}
