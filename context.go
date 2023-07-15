package di

import "context"

type contextKey struct{}

var containerKey contextKey = contextKey{}

// ContextWithContainer returns a new Context that carries the provided Container.
func ContextWithContainer(ctx context.Context, c Container) context.Context {
	return context.WithValue(ctx, containerKey, c)
}

// FromContext returns the Scope stored in ctx, if it exists.
func FromContext(ctx context.Context) (Container, bool) {
	s, ok := ctx.Value(containerKey).(Container)
	return s, ok
}
