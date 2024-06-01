package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/dicontext"
	"github.com/johnrutherford/di-kit/dihttp"
	"github.com/johnrutherford/di-kit/examples/bar"
	"github.com/johnrutherford/di-kit/examples/foo"
)

func Example_HTTP() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	c, err := di.NewContainer(
		di.Register(logger),
		di.Register(foo.NewFooService),
		di.Register(bar.NewBarService, di.Scoped),
	)
	if err != nil {
		return err
	}

	scopeMiddleware := dihttp.NewScopeMiddleware(
		dihttp.WithParent(c),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svc := dicontext.MustResolve[*bar.BarService](r.Context())
		svc.HandleRequest(r, w)
	})

	mux := http.NewServeMux()
	mux.Handle("/", scopeMiddleware(handler))

	http.ListenAndServe(":8080", nil)
	return nil
}
