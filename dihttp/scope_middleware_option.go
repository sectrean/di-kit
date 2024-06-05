package dihttp

import (
	"github.com/johnrutherford/di-kit"
)

// ScopeMiddlewareOption is an option used to configure the scope middleware when calling [NewScopeMiddleware].
type ScopeMiddlewareOption interface {
	applyScopeMiddleware(*scopeMiddleware)
}

type scopeMiddlewareOption func(*scopeMiddleware)

func (o scopeMiddlewareOption) applyScopeMiddleware(m *scopeMiddleware) {
	o(m)
}

// WithScopeOptions sets the options to use when calling [di.Container.NewScope] for each request.
func WithScopeOptions(opts ...di.ContainerOption) ScopeMiddlewareOption {
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
