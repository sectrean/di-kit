package di

import (
	"context"
	"reflect"
	"sync/atomic"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// A Scope allows you to resolve services.
//
// Scope can be used as a parameter for a service's constructor function.
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
//		return &DBFactory{scope}
//	}
//
//	func (f *DBFactory) NewDB(ctx context.Context, dbName string) *DB {
//		// Use the Scope to resolve dependencies...
//	}
//
// Scope is implemented by *Container.
type Scope interface {
	// Contains returns true if the Scope can resolve a service of the given type.
	//
	// Available options:
	// 	- [WithTag] specifies the tag associated with the service.
	Contains(t reflect.Type, opts ...ResolveOption) bool

	// Resolve returns a service of the given type from the Scope.
	//
	// Available options:
	// 	- [WithTag] specifies the tag associated with the service.
	Resolve(ctx context.Context, t reflect.Type, opts ...ResolveOption) (any, error)
}

// Resolve a service of type Service from the [Scope].
//
// See [Scope.Resolve] for more information.
func Resolve[Service any](ctx context.Context, s Scope, opts ...ResolveOption) (Service, error) {
	var val Service
	anyVal, err := s.Resolve(ctx, reflect.TypeFor[Service](), opts...)
	if anyVal != nil {
		val = anyVal.(Service)
	}

	return val, err
}

// MustResolve resolves a service of type Service from the [Scope].
//
// See [Scope.Resolve] for more information.
//
// This will panic if the service cannot be resolved.
func MustResolve[Service any](ctx context.Context, s Scope, opts ...ResolveOption) Service {
	val, err := Resolve[Service](ctx, s, opts...)
	if err != nil {
		panic(err)
	}
	return val
}

func newInjectedScope(s Scope, key serviceKey) (scope *injectedScope, ready func()) {
	wrapper := &injectedScope{
		scope: s,
		key:   key,
	}

	return wrapper, wrapper.setReady
}

// injectedScope wraps a Container to be injected as a Scope dependency.
// This is used to prevent the Scope from being used until the constructor function has returned.
// Otherwise a dependency cycle is possible.
type injectedScope struct {
	scope Scope

	// key is the service the Scope is getting injected into
	key   serviceKey
	ready atomic.Bool
}

func (s *injectedScope) setReady() {
	s.ready.Store(true)
}

func (s *injectedScope) Contains(t reflect.Type, opts ...ResolveOption) bool {
	return s.scope.Contains(t, opts...)
}

func (s *injectedScope) Resolve(
	ctx context.Context,
	t reflect.Type,
	opts ...ResolveOption,
) (any, error) {
	// Resolve cannot be called until the constructor function has returned.
	// Otherwise a deadlock is possible.
	if !s.ready.Load() {
		return nil, errors.Errorf(
			"di.Container.Resolve %v: "+
				"not supported within service constructor function",
			t,
		)
	}

	return s.scope.Resolve(ctx, t, opts...)
}

var _ Scope = (*injectedScope)(nil)
