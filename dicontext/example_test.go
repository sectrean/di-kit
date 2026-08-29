package dicontext_test

import (
	"context"
	"log/slog"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/dicontext"
	"github.com/sectrean/di-kit/examples/service"
)

func Example() {
	c, err := di.NewContainer(
		di.WithService(service.NewService),
	)
	if err != nil {
		slog.Error("container creation failed", "error", err)
		return
	}
	defer func() {
		closeErr := c.Close(context.Background())
		if closeErr != nil {
			slog.Error("container close failed", "error", err)
		}
	}()

	handleRequest := func(ctx context.Context) error {
		svc, err := dicontext.Resolve[service.Service](ctx)
		if err != nil {
			return err
		}

		return svc.Handle(ctx)
	}

	ctx := dicontext.WithScope(context.Background(), c)
	if err := handleRequest(ctx); err != nil {
		slog.Error("handle request failed", "error", err)
	}
}
