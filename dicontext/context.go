package dicontext

import (
	"context"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/internal/errors"
)

type scopeContextKey struct{}

// WithScope returns a new [context.Context] that carries the provided [di.Scope].
func WithScope(ctx context.Context, s di.Scope) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, s)
}

// Scope returns the [di.Scope] stored on the Context, if present.
func Scope(ctx context.Context) di.Scope {
	if s, ok := ctx.Value(scopeContextKey{}).(di.Scope); ok {
		return s
	}
	return nil
}

// Resolve resolves a service of the given type from the [di.Scope] stored on the
// [context.Context].
func Resolve[T any](ctx context.Context, opts ...di.ResolveOption) (T, error) {
	s := Scope(ctx)
	if s == nil {
		var val T
		err := errors.Errorf("resolve %s from context: scope not found on context", di.TypeOf[T]())
		return val, err
	}

	return di.Resolve[T](ctx, s, opts...)
}

// MustResolve resolves a service of the given type from the [di.Scope] stored on the
// [context.Context].
func MustResolve[T any](ctx context.Context, opts ...di.ResolveOption) T {
	s := Scope(ctx)
	if s == nil {
		panic("scope not found on context")
	}

	return di.MustResolve[T](ctx, s, opts...)
}
