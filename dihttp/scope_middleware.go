package dihttp

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/dicontext"
)

// Middleware is a function that wraps an HTTP handler.
type Middleware = func(http.Handler) http.Handler

// NewRequestScopeMiddleware returns HTTP middleware that creates a new child container by calling
// [di.Container.NewScope] for each request.
// The child container is stored on the request context and can be accessed using [dicontext.Scope], [dicontext.Resolve], or [dicontext.MustResolve].
// The child container is closed after the request is processed.
//
// Available options:
//   - [WithRequestService]: Register the current [*http.Request] with the request scope so it can be used as a dependency for scoped services.
//   - [WithContainerOptions]: Set [di.ContainerOption]s to use when creating each request scope.
//   - [WithNewScopeErrorHandler]: Set the error handler for when there is an error creating a new scope.
//   - [WithScopeCloseErrorHandler]: Set the error handler for when there is an error closing the scope.
//
// This will panic if parent is nil.
func NewRequestScopeMiddleware(parent *di.Container, opts ...ScopeMiddlewareOption) Middleware {
	if parent == nil {
		panic("dihttp.NewRequestScopeMiddleware: parent is nil")
	}

	return func(next http.Handler) http.Handler {
		mw := &scopeMiddleware{
			next:               next,
			parent:             parent,
			newScopeErrHandler: defaultNewScopeErrorHandler,
			closeErrHandler:    defaultScopeCloseErrorHandler,
		}

		for _, opt := range opts {
			opt.applyScopeMiddleware(mw)
		}

		return mw
	}
}

// NewScopeErrorHandler is a function that writes an error response to the client.
// This is called by the scope middleware when there is an error creating a new request-scoped [di.Container].
//
// The default handler logs the error to [slog.Default] and writes a "500 Internal Server Error" response.
type NewScopeErrorHandler = func(http.ResponseWriter, *http.Request, error)

func defaultNewScopeErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(),
		"error creating new di.Container scope for HTTP request",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"error", err,
	)

	w.WriteHeader(http.StatusInternalServerError)
}

// ScopeCloseErrorHandler is a function that handles errors when closing the request-scoped [di.Container]
// after the request has completed.
//
// The default handler logs the error to [slog.Default].
type ScopeCloseErrorHandler = func(*http.Request, error)

func defaultScopeCloseErrorHandler(r *http.Request, err error) {
	slog.ErrorContext(r.Context(),
		"error closing di.Container scope for HTTP request",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"error", err,
	)
}

type scopeMiddleware struct {
	next               http.Handler
	parent             *di.Container
	newScopeErrHandler NewScopeErrorHandler
	closeErrHandler    ScopeCloseErrorHandler
	opts               []di.ContainerOption
	registerRequest    bool
	closeTimeout       time.Duration
}

func (m *scopeMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	opts := m.opts

	// Register the current HTTP request with the scope if requested, so it can be
	// injected into scoped services.
	if m.registerRequest {
		opts = make([]di.ContainerOption, len(m.opts)+1)
		copy(opts, m.opts)
		opts[len(m.opts)] = di.WithService(r)
	}

	// Create child scope for the request
	child, newScopeErr := m.parent.NewScope(opts...)
	if newScopeErr != nil {
		m.newScopeErrHandler(w, r, newScopeErr)
		return
	}

	// Close the scope when the request is done
	defer func() {
		closeCtx := context.WithoutCancel(r.Context())
		if m.closeTimeout > 0 {
			var cancel context.CancelFunc
			closeCtx, cancel = context.WithTimeout(closeCtx, m.closeTimeout)
			defer cancel()
		}

		closeErr := child.Close(closeCtx)
		if closeErr != nil {
			m.closeErrHandler(r, closeErr)
		}
	}()

	// Add the scope to the request context
	scopeCtx := dicontext.WithScope(r.Context(), child)

	// Call the next handler with the new context
	m.next.ServeHTTP(w, r.WithContext(scopeCtx))
}
