package workflow

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

// Error marks a workflow step failure with stable operator-facing metadata.
// It implements the scheduler's OpError carrier interface, so returning it from
// an Executor causes the existing scheduler failure path to persist the embedded
// model.OpError.
type Error struct {
	Code      string
	Message   string
	Retryable bool
	Details   json.RawMessage
	Cause     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Code
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *Error) OpError() model.OpError {
	if e == nil {
		return model.OpError{}
	}
	message := e.Message
	if message == "" && e.Cause != nil {
		message = e.Cause.Error()
	}
	if message == "" {
		message = e.Code
	}
	code := e.Code
	if code == "" {
		code = "workflow_error"
	}
	return model.OpError{
		Code:       code,
		Message:    message,
		Retryable:  e.Retryable,
		Details:    e.Details,
		OccurredAt: time.Now().UTC(),
	}
}

// Retryable wraps err as a retryable workflow step error. The stable code is
// used for metrics, runtime events, and operator filtering.
func Retryable(code string, err error) error {
	return stepError(code, err, true)
}

// Permanent wraps err as a non-retryable workflow step error.
func Permanent(code string, err error) error {
	return stepError(code, err, false)
}

func stepError(code string, err error, retryable bool) error {
	if err == nil {
		err = errors.New(code)
	}
	return &Error{Code: code, Message: err.Error(), Retryable: retryable, Cause: err}
}
