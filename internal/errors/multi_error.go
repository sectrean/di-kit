package errors

import (
	stderrors "errors"
)

// MultiError is a collection of errors.
type MultiError []error

// Append appends an error to the collection.
func (e MultiError) Append(err error) MultiError {
	if err == nil {
		return e
	}
	return append(e, err)
}

// Join combines all errors into a single error.
func (e MultiError) Join() error {
	if len(e) == 0 {
		return nil
	}
	return stderrors.Join(e...)
}

// Wrap joins errors and then wraps the joined error with a message.
//
// Returns nil if there are no errors.
func (e MultiError) Wrap(msg string) error {
	if len(e) == 0 {
		return nil
	}
	return Wrap(e.Join(), msg)
}

// Wrapf joins errors and then wraps the joined error with a formatted message.
//
// Returns nil if there are no errors.
func (e MultiError) Wrapf(msg string, args ...any) error {
	if len(e) == 0 {
		return nil
	}
	return Wrapf(e.Join(), msg, args...)
}
