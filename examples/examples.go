package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/examples/foo"
)

func Example() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create the container
	c, err := di.NewContainer(
		// Register services with values and functions
		di.Register(logger),
		di.Register(foo.NewFooService),
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
	fooSvc := di.MustResolve[*foo.FooService](ctx, c)

	err = fooSvc.Run(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "error running service", "error", err)
	}
}
