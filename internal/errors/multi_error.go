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
