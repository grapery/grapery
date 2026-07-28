package domain

import "fmt"

const (
	FragmentGenerationErrorInvalidRequest = "invalid_request"
	FragmentGenerationErrorForbidden      = "forbidden"
	FragmentGenerationErrorNotFound       = "not_found"
	FragmentGenerationErrorConflict       = "conflict"
	FragmentGenerationErrorCancelled      = "cancelled"
	FragmentGenerationErrorInsufficient   = "insufficient_quota"
)

// FragmentGenerationError carries a stable code so clients and the agent can
// choose a recoverable UX instead of treating every failure as a server error.
type FragmentGenerationError struct {
	Code    string
	Message string
	Err     error
}

func (e *FragmentGenerationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *FragmentGenerationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewFragmentGenerationError(code, message string, err error) error {
	return &FragmentGenerationError{Code: code, Message: message, Err: err}
}
