package dicontext

import (
	"context"
	"reflect"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/errors"
)

type scopeContextKey struct{}

// WithScope returns a new [context.Context] that carries the provided [di.Scope].
func WithScope(ctx context.Context, s di.Scope) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, s)
}

// Scope returns the [di.Scope] stored on the [context.Context], if present.
func Scope(ctx context.Context) di.Scope {
	if s, ok := ctx.Value(scopeContextKey{}).(di.Scope); ok {
		return s
	}
	return nil
}

// Resolve a service of type Service from the [di.Scope] stored on the
// [context.Context].
//
// This will return an error if there is no [di.Scope] on the context, or the service cannot be resolved.
//
// See [di.Scope.Resolve] for more information.
func Resolve[Service any](ctx context.Context, opts ...di.ResolveOption) (Service, error) {
	var t = reflect.TypeFor[Service]()
	var val Service

	s := Scope(ctx)
	if s == nil {
		return val, errors.Errorf("resolve %s from context: scope not found on context", t)
	}

	anyVal, err := s.Resolve(ctx, t, opts...)
	if anyVal != nil {
		val = anyVal.(Service)
	}

	return val, errors.Wrap(err, "resolve from context")
}

// MustResolve resolves a service of type Service from the [di.Scope] stored on the
// [context.Context].
//
// This will panic if there is no [di.Scope] on the context, or the service cannot be resolved.
//
// See [di.Scope.Resolve] for more information.
func MustResolve[Service any](ctx context.Context, opts ...di.ResolveOption) Service {
	val, err := Resolve[Service](ctx, opts...)
	if err != nil {
		panic(err)
	}
	return val
}
