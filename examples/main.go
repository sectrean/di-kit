package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/examples/foo"
)

func run() (exitCode int) {
	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create the container
	c, err := di.NewContainer(
		// Register services with values and functions
		di.Register(logger),
		di.Register(foo.NewFooService),
	)
	if err != nil {
		logger.Error("Create Container", "error", err)
		return 1
	}

	// Close the container when we're done
	defer func() {
		err := c.Close(context.Background())
		if err != nil {
			logger.Error("Close Container", "error", err)
			exitCode = 1
		}
	}()

	// Resolve a service from the container
	ctx := context.Background()
	fooSvc := di.MustResolve[*foo.FooService](ctx, c)

	// Use the service
	err = fooSvc.Run(ctx)
	if err != nil {
		logger.Error("Run FooService", "error", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run())
}
