package dihttp

import (
	"time"

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

// WithRequestService registers the current [*http.Request] with each request scope so it can be
// resolved directly or injected as a dependency into scoped services.
//
// Registration is opt-in because it is not needed unless a scoped service depends on the request,
// and it adds a small amount of work to every request.
func WithRequestService() ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) {
		m.registerRequest = true
	})
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
			m.newScopeErrHandler = h
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
			m.closeErrHandler = h
		}
	})
}

// WithScopeCloseTimeout sets a timeout for closing the request scope at the end of each request.
//
// The default is no timeout.
func WithScopeCloseTimeout(timeout time.Duration) scopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) {
		m.closeTimeout = timeout
	})
}
