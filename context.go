package di

import (
	"context"
)

type scopeContextKey struct{}

// ContextWithScope returns a new Context that carries the provided Scope.
func ContextWithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, s)
}

// ScopeFromContext returns the Scope stored on the Context, if it exists.
func ScopeFromContext(ctx context.Context) Scope {
	if s, ok := ctx.Value(scopeContextKey{}).(Scope); ok {
		return s
	}
	return nil
}
