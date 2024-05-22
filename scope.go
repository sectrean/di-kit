package di

import (
	"context"
	"reflect"
)

// Scope allows you to resolve services.
//
// A Scope can be injected into functions to allow them to resolve services. However,
// it cannot be used within the constructor function. It can be stored in a struct or
// used in a closure after the constructor function has returned.
//
// Scope is implemented by *Container.
type Scope interface {
	// Contains returns true if the Scope has a service of the given type.
	//
	// Available options:
	// 	- [WithTag] specifies the tag associated with the service.
	Contains(t reflect.Type, opts ...ServiceOption) bool

	// Resolve returns a service of the given type from the Scope.
	//
	// Available options:
	// 	- [WithTag] specifies the tag associated with the service.
	Resolve(ctx context.Context, t reflect.Type, opts ...ServiceOption) (any, error)
}

// Resolve a service of the given type from the [Scope].
func Resolve[T any](ctx context.Context, s Scope, opts ...ServiceOption) (T, error) {
	var val T
	anyVal, err := s.Resolve(ctx, reflect.TypeFor[T](), opts...)
	if anyVal != nil {
		val = anyVal.(T)
	}

	return val, err
}

// MustResolve resolves a service of the given type from the [Scope].
//
// If the service cannot be resolved, this function will panic.
func MustResolve[T any](ctx context.Context, s Scope, opts ...ServiceOption) T {
	val, err := Resolve[T](ctx, s, opts...)
	if err != nil {
		panic(err)
	}
	return val
}
