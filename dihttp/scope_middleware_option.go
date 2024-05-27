package dihttp

import (
	"github.com/johnrutherford/di-kit"
)

// ScopeMiddlewareOption is an option used when calling [NewScopeMiddleware] to configure the scope middleware.
type ScopeMiddlewareOption interface {
	applyScopeMiddleware(*scopeMiddleware)
}

type scopeMiddlewareOption func(*scopeMiddleware)

func (o scopeMiddlewareOption) applyScopeMiddleware(m *scopeMiddleware) {
	o(m)
}

// WithParent sets the parent [di.Container] for new scopes.
func WithParent(parent *di.Container) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) {
		m.opts = append(m.opts, di.WithParent(parent))
	})
}

// WithContainerOptions sets additional [di.Container] options for new scopes.
// This can be used to register services or set other options on each new child scope.
func WithContainerOptions(opts ...di.ContainerOption) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) {
		m.opts = append(m.opts, opts...)
	})
}

// WithNewScopeErrorHandler sets the error handler for when there is an error creating a new scope.
func WithNewScopeErrorHandler(fn NewScopeErrorHandler) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) {
		m.newScopeHandler = fn
	})
}

// WithScopeCloseErrorHandler sets the error handler for when there is an error closing the scope.
func WithScopeCloseErrorHandler(fn ScopeCloseErrorHandler) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) {
		m.closeHandler = fn
	})
}
