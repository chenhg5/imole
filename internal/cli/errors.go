package cli

import "errors"

const (
	ExitSuccess  = 0
	ExitError    = 1
	ExitUsage    = 2  // invalid arguments / usage error
	ExitNotFound = 3  // resource not found
	ExitConflict = 5  // already exists
	ExitDryRun   = 10 // dry-run passed, safe to execute
)

type Error struct {
	Code       string `json:"error"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Retryable  bool   `json:"retryable"`
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(code, message, suggestion string, retryable bool) *Error {
	return &Error{Code: code, Message: message, Suggestion: suggestion, Retryable: retryable}
}

var (
	ErrUsage    = errors.New("invalid usage")
	ErrNotFound = errors.New("resource not found")
	ErrConflict = errors.New("conflict / already exists")
)
