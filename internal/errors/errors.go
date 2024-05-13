package errors

import (
	stderrors "errors"
	"fmt"
)

// New returns an error with the given message.
func New(msg string) error {
	return stderrors.New(msg)
}

// Errorf returns an error with the given message.
func Errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// Wrap returns an error with the given message and wraps the original error.
//
// Returns nil if the original error is nil.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", msg, err)
}

// Wrapf returns an error with a formatted message and wraps the original error.
//
// Returns nil if the original error is nil.
func Wrapf(err error, msg string, args ...any) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", fmt.Sprintf(msg, args...), err)
}
