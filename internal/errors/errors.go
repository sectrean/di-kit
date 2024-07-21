// Package errors provides some error utilities and helpers.
// We want to avoid dependencies on 3rd party packages for errors.
//
// This package does not add stack traces to errors.
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
func Errorf(format string, a ...any) error {
	return fmt.Errorf(format, a...)
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
func Wrapf(err error, format string, a ...any) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", fmt.Sprintf(format, a...), err)
}
