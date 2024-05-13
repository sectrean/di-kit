package di

import (
	stderrors "errors"
)

var (
	// ErrTypeNotRegistered is returned when a type is not registered.
	ErrTypeNotRegistered = stderrors.New("type not registered")

	// ErrDependencyCycle is returned when a dependency cycle is detected.
	ErrDependencyCycle = stderrors.New("dependency cycle detected")

	// ErrContainerClosed is returned when the container is closed.
	ErrContainerClosed = stderrors.New("container closed")
)
