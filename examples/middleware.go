package main

import (
	"log/slog"
	"net/http"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/dicontext"
)

func RequestScopeMiddleware(logger *slog.Logger, parent *di.Container) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Create a new scope for the request
			scope, err := di.NewContainer(
				di.WithParent(parent),
				// Register any request-specific services here
			)
			if err != nil {
				logger.ErrorContext(ctx, "Create Request Scope", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Add the scope to the request context
			ctx = dicontext.WithScope(ctx, scope)
			next.ServeHTTP(w, r.WithContext(ctx))

			err = scope.Close(ctx)
			if err != nil {
				logger.ErrorContext(ctx, "Close Request Scope", "error", err)
			}
		})
	}
}
