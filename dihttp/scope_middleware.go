package dihttp

import (
	"log/slog"
	"net/http"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/dicontext"
)

// NewScopeMiddleware creates a new middleware that creates a new [di.Scope] for each request.
// The scope is closed after the request has been processed.
//
// The scope is stored on the request context and can be accessed using [dicontext.Scope], [dicontext.Resolve], or [dicontext.MustResolve].
//
// Available options:
//   - WithParent: Set the parent [di.Container] for the new scope.
//   - WithContainerOptions: Set additional [di.Container] options.
//   - WithNewScopeErrorHandler: Set the error handler for when there is an error creating a new scope.
//   - WithScopeCloseErrorHandler: Set the error handler for when there is an error closing the scope.
func NewScopeMiddleware(opts ...ScopeMiddlewareOption) func(http.Handler) http.Handler {
	mw := &scopeMiddleware{
		newScopeHandler: defaultNewScopeErrorHandler,
		closeHandler:    defaultScopeCloseErrorHandler,
	}
	for _, opt := range opts {
		opt.applyScopeMiddleware(mw)
	}

	return func(next http.Handler) http.Handler {
		mw.next = next
		return mw
	}
}

// NewScopeErrorHandler is a function that writes an error response to the client.
// This is called by the scope middleware when there is an error creating the [di.Container].
//
// The default handler logs the error to [slog.Default()] and writes a 500 Internal Server Error response.
type NewScopeErrorHandler = func(w http.ResponseWriter, r *http.Request, err error)

func defaultNewScopeErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "error creating new HTTP request scope", "error", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// ScopeCloseErrorHandler is a function that handles errors when closing the [di.Container]
// after the request has completed.
//
// The default handler logs the error to [slog.Default()].
type ScopeCloseErrorHandler = func(r *http.Request, err error)

func defaultScopeCloseErrorHandler(r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "error closing HTTP request scope", "error", err)
}

type scopeMiddleware struct {
	newScopeHandler NewScopeErrorHandler
	closeHandler    ScopeCloseErrorHandler
	opts            []di.ContainerOption
	next            http.Handler
}

func (m *scopeMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	opts := append(m.opts,
		// Add the *http.Request to the scope
		di.Register(r),
	)

	scope, err := di.NewContainer(opts...)
	if err != nil {
		if m.newScopeHandler != nil {
			m.newScopeHandler(w, r, err)
		}
		return
	}

	ctx := dicontext.WithScope(r.Context(), scope)
	m.next.ServeHTTP(w, r.WithContext(ctx))

	err = scope.Close(ctx)
	if err != nil && m.closeHandler != nil {
		m.closeHandler(r, err)
	}
}
