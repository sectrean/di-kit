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

// Scope returns the [di.Scope] stored on the [context.Context], if present.
func Scope(ctx context.Context) di.Scope {
	if s, ok := ctx.Value(scopeContextKey{}).(di.Scope); ok {
		return s
	}
	return nil
}

// Resolve resolves a service of the given type from the [di.Scope] stored on the
// [context.Context].
func Resolve[T any](ctx context.Context, opts ...di.ResolveOption) (T, error) {
	var val T

	s := Scope(ctx)
	if s == nil {
		return val, errors.Errorf(
			"resolve %s from context: scope not found on context", di.TypeOf[T](),
		)
	}

	val, err := di.Resolve[T](ctx, s, opts...)
	return val, errors.Wrap(err, "resolve from context")
}

// MustResolve resolves a service of the given type from the [di.Scope] stored on the
// [context.Context].
func MustResolve[T any](ctx context.Context, opts ...di.ResolveOption) T {
	val, err := Resolve[T](ctx, opts...)
	if err != nil {
		panic(err)
	}
	return val
}
