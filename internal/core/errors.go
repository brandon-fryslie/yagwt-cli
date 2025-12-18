package core

import (
	"encoding/json"
	"fmt"
)

// ErrorCode represents a specific error condition
type ErrorCode string

const (
	ErrDirty     ErrorCode = "E_DIRTY"
	ErrNotFound  ErrorCode = "E_NOT_FOUND"
	ErrAmbiguous ErrorCode = "E_AMBIGUOUS"
	ErrGit       ErrorCode = "E_GIT"
	ErrPolicy    ErrorCode = "E_POLICY"
	ErrLocked    ErrorCode = "E_LOCKED"
	ErrBroken    ErrorCode = "E_BROKEN"
	ErrConflict  ErrorCode = "E_CONFLICT"
	ErrTimeout   ErrorCode = "E_TIMEOUT"
	ErrConfig    ErrorCode = "E_CONFIG"
)

// Error represents a structured error with hints
type Error struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	Hints   []Hint                 `json:"hints,omitempty"`
	Wrapped error                  `json:"-"` // Underlying error (not serialized directly)
}

// Hint provides an actionable suggestion
type Hint struct {
	Message string `json:"message"`
	Command string `json:"command,omitempty"`
}

func (e *Error) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Wrapped)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error for errors.Is and errors.As
func (e *Error) Unwrap() error {
	return e.Wrapped
}

// NewError creates a new Error with the given code and message
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// WrapError creates a new Error that wraps an underlying error
func WrapError(code ErrorCode, message string, wrapped error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
		Wrapped: wrapped,
	}
}

// WithDetail adds a detail to the error
func (e *Error) WithDetail(key string, value interface{}) *Error {
	e.Details[key] = value
	return e
}

// WithHint adds a hint to the error
func (e *Error) WithHint(message string, command string) *Error {
	e.Hints = append(e.Hints, Hint{Message: message, Command: command})
	return e
}

// Exit code mapping for CLI
func (e *Error) ExitCode() int {
	switch e.Code {
	case ErrDirty:
		return 3
	case ErrNotFound:
		return 5
	case ErrAmbiguous:
		return 2
	case ErrLocked:
		return 3
	default:
		return 1
	}
}

// MarshalJSON implements json.Marshaler to include wrapped error message in JSON
func (e *Error) MarshalJSON() ([]byte, error) {
	type Alias Error
	aux := &struct {
		WrappedError string `json:"wrappedError,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if e.Wrapped != nil {
		aux.WrappedError = e.Wrapped.Error()
	}
	return json.Marshal(aux)
}
