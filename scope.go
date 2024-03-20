package di

import (
	"context"
	"reflect"
)

// Scope allows you to resolve services and invoke functions with injected dependencies.
//
// A Scope can be injected into functions to allow them to resolve services. However,
// it cannot be used within the constructor function. It can be stored in a struct or
// used in a closure after the constructor function has returned.
type Scope interface {
	// HasType returns true if the scope has a service of the given type
	HasType(typ reflect.Type) bool

	// Resolve returns a service of the given type
	Resolve(ctx context.Context, typ reflect.Type) (any, error)

	// Invoke calls the given function with dependencies injected from the container
	Invoke(ctx context.Context, fn any) error
}

// Resolve a service of the given type from the [Scope].
func Resolve[T any](ctx context.Context, s Scope) (T, error) {
	var val T

	// Resolve the service
	anyVal, err := s.Resolve(ctx, TypeOf[T]())
	if anyVal != nil {
		val = anyVal.(T)
	}

	return val, err
}

// MustResolve resolves a service of the given type from the [Scope].
//
// If the service cannot be resolved, this function panics.
func MustResolve[T any](ctx context.Context, s Scope) T {
	val, err := Resolve[T](ctx, s)
	if err != nil {
		panic(err)
	}
	return val
}

// HasType returns true if the [Scope] has a service of the given type.
func HasType[T any](s Scope) bool {
	return s.HasType(TypeOf[T]())
}
