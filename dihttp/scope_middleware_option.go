package dihttp

import (
	"errors"

	"github.com/johnrutherford/di-kit"
)

// ScopeMiddlewareOption is an option used to configure the scope middleware when calling [NewRequestScopeMiddleware].
type ScopeMiddlewareOption interface {
	applyScopeMiddleware(*scopeMiddleware) error
}

type scopeMiddlewareOption func(*scopeMiddleware) error

func (o scopeMiddlewareOption) applyScopeMiddleware(m *scopeMiddleware) error {
	return o(m)
}

// WithContainerOptions sets the options to use when calling [di.Container.NewScope] for each request.
func WithContainerOptions(opts ...di.ContainerOption) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) error {
		m.opts = append(m.opts, opts...)
		return nil
	})
}

// WithNewScopeErrorHandler sets the error handler for when there is an error creating a new scope.
//
// The default handler logs the error to [slog.Default] and writes a "500 Internal Server Error" response.
// This will panic if h is nil.
func WithNewScopeErrorHandler(h NewScopeErrorHandler) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) error {
		if h == nil {
			return errors.New("WithNewScopeErrorHandler: h is nil")
		}

		m.newScopeHandler = h
		return nil
	})
}

// WithScopeCloseErrorHandler sets the error handler for when there is an error closing the
// request-scoped [di.Container] after the request has completed.
//
// The default handler logs the error to [slog.Default].
// This will panic if h is nil.
func WithScopeCloseErrorHandler(h ScopeCloseErrorHandler) ScopeMiddlewareOption {
	return scopeMiddlewareOption(func(m *scopeMiddleware) error {
		if h == nil {
			return errors.New("WithScopeCloseErrorHandler: h is nil")
		}

		m.closeHandler = h
		return nil
	})
}
