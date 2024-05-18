package di

import (
	"context"
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
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
	Contains(t reflect.Type, opts ...ContainsOption) bool

	// Resolve returns a service of the given type from the Scope.
	//
	// Available options:
	// 	- [WithTag] specifies the tag associated with the service.
	Resolve(ctx context.Context, t reflect.Type, opts ...ResolveOption) (any, error)
}

// Resolve a service of the given type from the [Scope].
func Resolve[T any](ctx context.Context, s Scope, opts ...ResolveOption) (T, error) {
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
func MustResolve[T any](ctx context.Context, s Scope, opts ...ResolveOption) T {
	val, err := Resolve[T](ctx, s, opts...)
	if err != nil {
		panic(err)
	}
	return val
}

// Invoke calls the given function with dependencies resolved from the provided Scope.
//
// The function may take any number of arguments. These dependencies must be registered with the Scope.
// The function may also accept a context.Context.
// The function may return an error. Any other return values are ignored.
func Invoke(ctx context.Context, s Scope, fn any) error {
	fnType := reflect.TypeOf(fn)
	fnVal := reflect.ValueOf(fn)

	// Make sure fn is a function
	if fnType.Kind() != reflect.Func {
		return errors.Errorf("invoke fn %T: fn must be a function", fn)
	}

	// Resolve fn arguments from the Scope
	// Stop at the first error
	in := make([]reflect.Value, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		argType := fnType.In(i)
		argVal, argErr := s.Resolve(ctx, argType)
		if argErr != nil {
			return errors.Wrapf(argErr, "invoke fn %T", fn)
		}
		in[i] = reflect.ValueOf(argVal)
	}

	// Check for a context error before invoking the function
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Invoke the function
	out := fnVal.Call(in)

	// See if the function returns an error
	for i := 0; i < fnType.NumOut(); i++ {
		if fnType.Out(i) == errorType {
			err, _ := out[i].Interface().(error)
			return err
		}
	}

	return nil
}
