package di

import (
	"context"

	"reflect"
	"sync/atomic"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// A Scope allows you to resolve services.
//
// Scope can be used as an argument for a service's constructor function.
// This can be used to create a "factory" service.
//
// Note that the Scope should be stored on the service struct for later use.
// [Scope.Resolve] cannot be called from within the constructor function.
// It will return an error.
//
// Example:
//
//	type DBFactory struct {
//		scope di.Scope
//	}
//
//	func NewDBFactory(scope di.Scope) *DBFactory {
//		return &DBFactory{scope: scope}
//	}
//
//	func (f *DBFactory) NewDB(string dbName) *DB {
//		// Use the Scope to resolve dependencies...
//	}
//
// Scope is implemented by *Container.
type Scope interface {
	// Contains returns whether the Scope can resolve a service of the given type.
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
//
// See [Scope.Resolve] for more information.
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
// See [Scope.Resolve] for more information.
//
// This will panic if the service cannot be resolved.
func MustResolve[T any](ctx context.Context, s Scope, opts ...ServiceOption) T {
	val, err := Resolve[T](ctx, s, opts...)
	if err != nil {
		panic(err)
	}
	return val
}

func newInjectedScope(key serviceKey, s Scope) (*injectedScope, func()) {
	wrapper := &injectedScope{
		key:   key,
		scope: s,
	}

	return wrapper, wrapper.setReady
}

// injectedScope wraps a Container to be injected as a Scope dependency.
type injectedScope struct {
	// key is the service the Scope is getting injected into
	key   serviceKey
	scope Scope
	ready atomic.Bool
}

func (s *injectedScope) setReady() {
	s.ready.Store(true)
}

func (s *injectedScope) Contains(t reflect.Type, opts ...ServiceOption) bool {
	return s.scope.Contains(t, opts...)
}

func (s *injectedScope) Resolve(
	ctx context.Context,
	t reflect.Type,
	opts ...ServiceOption,
) (any, error) {
	if !s.ready.Load() {
		return nil, errors.Errorf(
			"resolve %v: "+
				"resolve not supported on di.Scope while resolving %s: "+
				"the scope must be stored and used later",
			t, s.key,
		)
	}

	return s.scope.Resolve(ctx, t, opts...)
}

var _ Scope = (*injectedScope)(nil)
