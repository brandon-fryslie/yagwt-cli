package core

import "github.com/bmf/yagwt/internal/errors"

// Re-export error types from errors package to maintain API compatibility
type (
	ErrorCode = errors.ErrorCode
	Error     = errors.Error
	Hint      = errors.Hint
)

// Re-export error codes
const (
	ErrDirty     = errors.ErrDirty
	ErrNotFound  = errors.ErrNotFound
	ErrAmbiguous = errors.ErrAmbiguous
	ErrGit       = errors.ErrGit
	ErrPolicy    = errors.ErrPolicy
	ErrLocked    = errors.ErrLocked
	ErrBroken    = errors.ErrBroken
	ErrConflict  = errors.ErrConflict
	ErrTimeout   = errors.ErrTimeout
	ErrConfig    = errors.ErrConfig
)

// Re-export error constructors
var (
	NewError  = errors.NewError
	WrapError = errors.WrapError
)
