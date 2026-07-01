package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/examples/service"
)

func main() {
	ctx := context.Background()
	logger := NewLogger()

	// Create the container
	c, err := di.NewContainer(
		// Register services with values and functions
		di.WithService(logger),
		di.WithService(service.NewService),
	)
	if err != nil {
		logger.ErrorContext(ctx, "error creating container", "error", err)
		return
	}

	// Close the container when done
	defer func() {
		err := c.Close(ctx)
		if err != nil {
			logger.ErrorContext(ctx, "error closing container", "error", err)
		}
	}()

	// Resolve our service from the container
	svc := di.MustResolve[*service.Service](ctx, c)

	err = svc.Run(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "error running service", "error", err)
	}
}

func NewLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
