package model

import "errors"

// Sentinel errors for the example domain.
// Use errors.Is() to check for these in handlers and tests.
var (
	ErrNotFound          = errors.New("example not found")
	ErrAlreadyExists     = errors.New("example already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidStatus     = errors.New("invalid status transition")
	ErrNameRequired      = errors.New("name is required")
	ErrNameTooLong       = errors.New("name exceeds maximum length")
	ErrDescriptionToLong = errors.New("description exceeds maximum length")
	ErrDatabaseError     = errors.New("database operation failed")
	ErrCacheError        = errors.New("cache operation failed")
	ErrTimeout           = errors.New("operation timed out")
)
