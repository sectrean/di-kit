package dihttp

import (
	"log/slog"
	"net/http"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/dicontext"
)

// RequestScopeMiddleware returns HTTP middleware that creates a new child container by calling
// [di.Container.NewScope] for each request.
// The child container is stored on the request context and can be accessed using [dicontext.Scope], [dicontext.Resolve], or [dicontext.MustResolve].
// The child container is closed after the request is processed.
//
// The current [*http.Request] is automatically registered with the child-scoped container. It can be used as a dependency for scoped services.
//
// Available options:
//   - WithScopeOptions: Set [di.ContainerOptions]s options to use when creating each request scope.
//   - WithNewScopeErrorHandler: Set the error handler for when there is an error creating a new scope.
//   - WithScopeCloseErrorHandler: Set the error handler for when there is an error closing the scope.
//
// This will panic if parent is nil.
func RequestScopeMiddleware(parent *di.Container, opts ...ScopeMiddlewareOption) func(http.Handler) http.Handler {
	if parent == nil {
		panic("parent is nil")
	}

	return func(next http.Handler) http.Handler {
		mw := &scopeMiddleware{
			next:            next,
			parent:          parent,
			newScopeHandler: defaultNewScopeErrorHandler,
			closeHandler:    defaultScopeCloseErrorHandler,
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
		"error creating new Container scope for HTTP request",
		"error", err,
		"request", r,
	)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// ScopeCloseErrorHandler is a function that handles errors when closing the request-scoped [di.Container]
// after the request has completed.
//
// The default handler logs the error to [slog.Default].
type ScopeCloseErrorHandler = func(*http.Request, error)

func defaultScopeCloseErrorHandler(r *http.Request, err error) {
	slog.ErrorContext(r.Context(),
		"error closing Container scope for HTTP request",
		"error", err,
		"request", r,
	)
}

type scopeMiddleware struct {
	next            http.Handler
	parent          *di.Container
	opts            []di.ContainerOption
	newScopeHandler NewScopeErrorHandler
	closeHandler    ScopeCloseErrorHandler
}

func (m *scopeMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	opts := append(m.opts,
		// Register the *http.Request with the new scope
		di.WithService(r),
	)

	// Create child scope for the request
	scope, err := m.parent.NewScope(opts...)
	if err != nil {
		m.newScopeHandler(w, r, err)
		return
	}

	// Add the scope to the request context
	// Call the next handler with the new context
	ctx := dicontext.WithScope(r.Context(), scope)
	m.next.ServeHTTP(w, r.WithContext(ctx))

	// Close the scope after the request has been processed
	err = scope.Close(ctx)
	if err != nil {
		m.closeHandler(r, err)
	}
}
