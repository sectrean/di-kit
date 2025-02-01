package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/dicontext"
	"github.com/sectrean/di-kit/dihttp"
	"github.com/sectrean/di-kit/examples/bar"
	"github.com/sectrean/di-kit/examples/foo"
)

func HTTP_Example() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	c, err := di.NewContainer(
		di.WithService(logger),
		di.WithService(foo.NewFooService),
		di.WithService(bar.NewBarService, di.ScopedLifetime),
	)
	if err != nil {
		logger.Error("error creating container", "error", err)
		return
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svc := dicontext.MustResolve[*bar.BarService](r.Context())
		svc.HandleRequest(r, w)
	})

	scopeMiddleware, err := dihttp.NewRequestScopeMiddleware(c)
	if err != nil {
		logger.Error("error creating scope middleware", "error", err)
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/", scopeMiddleware(handler))

	err = http.ListenAndServe(":8080", nil)
	if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("http server error", "error", err)
		return
	}

	logger.Info("server stopped")
}
