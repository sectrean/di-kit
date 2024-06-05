package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/dicontext"
	"github.com/johnrutherford/di-kit/dihttp"
	"github.com/johnrutherford/di-kit/examples/bar"
	"github.com/johnrutherford/di-kit/examples/foo"
)

func HTTP_Example() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	c, err := di.NewContainer(
		di.Register(logger),
		di.Register(foo.NewFooService),
		di.Register(bar.NewBarService, di.Scoped),
	)
	if err != nil {
		logger.Error("error creating container", "error", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svc := dicontext.MustResolve[*bar.BarService](r.Context())
		svc.HandleRequest(r, w)
	})

	mux := http.NewServeMux()

	scopeMiddleware := dihttp.NewScopeMiddleware(c)
	mux.Handle("/", scopeMiddleware(handler))

	err = http.ListenAndServe(":8080", nil)
	if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("http server error", "error", err)
		return
	}

	logger.Info("server stopped")
}
