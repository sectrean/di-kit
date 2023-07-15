package di

import "errors"

var (
	ErrTypeNotRegistered  = errors.New("type not registered")
	ErrContainerClosed    = errors.New("container closed")
	ErrCircularDependency = errors.New("circular dependency")
)
