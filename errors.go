package di

import (
	"errors"
)

var (
	// ErrTypeNotRegistered is returned when a type is not registered.
	ErrTypeNotRegistered = errors.New("type not registered")
	// ErrDependencyCycle is returned when a dependency cycle is detected.
	ErrDependencyCycle = errors.New("dependency cycle detected")
)
