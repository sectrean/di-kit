package examples

import (
	"context"
	"log/slog"
	"os"

	"github.com/johnrutherford/di-kit"
	"github.com/johnrutherford/di-kit/examples/foo"
)

func Example() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create the container
	c, err := di.NewContainer(
		// Register services with values and functions
		di.Register(logger),
		di.Register(foo.NewFooService),
	)
	if err != nil {
		return err
	}

	// Resolve a service from the container
	ctx := context.Background()
	fooSvc := di.MustResolve[*foo.FooService](ctx, c)

	// Use the service
	err = fooSvc.Run(ctx)
	if err != nil {
		return err
	}

	// Close the container
	err = c.Close(ctx)
	if err != nil {
		return err
	}

	return nil
}
