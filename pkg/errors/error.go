package errors

import (
	"fmt"
)

type Error struct {
	HTTPCode int
	Code     int
	Message  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("ERROR(%d): %s", e.Code, e.Message)
}

var (
	ErrNotFound = &Error{
		HTTPCode: 404,
		Code:     404,
		Message:  "object could not be found.",
	}
	ErrDuplicate = &Error{
		HTTPCode: 409,
		Code:     409001,
		Message:  "object already exists.",
	}
	ErrMultipleRecords = &Error{
		HTTPCode: 409,
		Code:     409002,
		Message:  "found multiple records when expected to find one.",
	}
	ErrGenerationConflict = &Error{
		HTTPCode: 409,
		Code:     409003,
		Message:  "encountered generation conflict.",
	}
	ErrUnknown = &Error{
		HTTPCode: 500,
		Code:     500,
		Message:  "unknown error.",
	}
)
