package dihttp

import (
	"log/slog"
	"net/http"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/dicontext"
	"github.com/sectrean/di-kit/internal/errors"
)

// Middleware is a function that wraps an HTTP handler.
type Middleware = func(http.Handler) http.Handler

// NewRequestScopeMiddleware returns HTTP middleware that creates a new child container by calling
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
func NewRequestScopeMiddleware(parent *di.Container, opts ...ScopeMiddlewareOption) (Middleware, error) {
	if parent == nil {
		return nil, errors.New("dihttp.NewRequestScopeMiddleware: parent is nil")
	}

	mw := &scopeMiddleware{
		parent:          parent,
		newScopeHandler: defaultNewScopeErrorHandler,
		closeHandler:    defaultScopeCloseErrorHandler,
	}

	var errs []error
	for _, opt := range opts {
		err := opt.applyScopeMiddleware(mw)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, errors.Wrap(err, "dihttp.NewRequestScopeMiddleware")
	}

	return func(next http.Handler) http.Handler {
		h := *mw
		h.next = next

		return h
	}, nil
}

// NewScopeErrorHandler is a function that writes an error response to the client.
// This is called by the scope middleware when there is an error creating a new request-scoped [di.Container].
//
// The default handler logs the error to [slog.Default] and writes a "500 Internal Server Error" response.
type NewScopeErrorHandler = func(http.ResponseWriter, *http.Request, error)

func defaultNewScopeErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(),
		"error creating new di.Container scope for HTTP request",
		"error", err,
		"request", r,
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
		"error", err,
		"request", r,
	)
}

type scopeMiddleware struct {
	next            http.Handler
	parent          *di.Container
	newScopeHandler NewScopeErrorHandler
	closeHandler    ScopeCloseErrorHandler
	opts            []di.ContainerOption
}

func (m scopeMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Use provided options and also register the current HTTP request
	opts := make([]di.ContainerOption, len(m.opts)+1)
	copy(opts, m.opts)
	opts[len(m.opts)] = di.WithService(r)

	// Create child scope for the request
	scope, err := m.parent.NewScope(opts...)
	if err != nil {
		m.newScopeHandler(w, r, err)
		return
	}

	// Add the scope to the request context
	ctx := dicontext.WithScope(r.Context(), scope)

	// Call the next handler with the new context
	m.next.ServeHTTP(w, r.WithContext(ctx))

	// Close the scope after the request has been processed
	err = scope.Close(ctx)
	if err != nil {
		m.closeHandler(r, err)
	}
}
