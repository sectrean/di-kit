package dihttp

import (
	"github.com/sectrean/di-kit"
)

// ScopeMiddlewareOption is an option used to configure the scope middleware when calling [NewRequestScopeMiddleware].
type ScopeMiddlewareOption interface {
	applyScopeMiddleware(*scopeMiddleware)
}

type scopeMiddlewareOption func(*scopeMiddleware)

func (o scopeMiddlewareOption) applyScopeMiddleware(m *scopeMiddleware) {
	o(m)
}

// WithContainerOptions sets the options to use when calling [di.Container.NewScope] for each request.
func WithContainerOptions(opts ...di.ContainerOption) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) {
		m.opts = append(m.opts, opts...)
	})
}

// WithNewScopeErrorHandler sets the error handler for when there is an error creating a new scope.
//
// The default handler logs the error to [slog.Default] and writes a "500 Internal Server Error" response.
func WithNewScopeErrorHandler(h NewScopeErrorHandler) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) {
		if h != nil {
			m.newScopeHandler = h
		}
	})
}

// WithScopeCloseErrorHandler sets the error handler for when there is an error closing the
// request-scoped [di.Container] after the request has completed.
//
// The default handler logs the error to [slog.Default].
func WithScopeCloseErrorHandler(h ScopeCloseErrorHandler) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) {
		if h != nil {
			m.closeHandler = h
		}
	})
}
