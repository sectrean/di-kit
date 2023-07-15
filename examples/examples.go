package examples

import (
	"context"
	"log"

	"github.com/johnrutherford/di-gen"
	"github.com/johnrutherford/di-gen/examples/foo"
)

func Basic() {
	// Create the container
	// Provide functions and values to the container
	c, err := di.NewContainer(
		di.Provide(foo.NewFooService),
	)
	if err != nil {
		log.Fatalf("error creating container: %+v", err)
	}

	ctx := context.Background()
	// Invoke a function and inject services from the container
	err = c.Invoke(ctx, func(ctx context.Context, foo foo.FooService) {
		foo.DoFoo()
	})
	if err != nil {
		log.Fatalf("error invoking function: %+v", err)
	}

	// Close the container when you're done
	err = c.Close(ctx)
	if err != nil {
		log.Fatalf("error closing container: %+v", err)
	}
}

func Scopes() {

}
