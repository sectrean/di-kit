/*
Package dicontext provides utilities for working with [di.Scope] and [context.Context].

Example:

	c, err := di.NewContainer(di.WithModule(Dependencies))
	...

	// Add the Container to the context
	ctx := context.Background()
	ctx = dicontext.WithScope(ctx, c)
	...

	// Resolve dependencies using the container scope on the context
	svc, err := dicontext.Resolve[MyService](ctx)
	...
*/
package dicontext

import "github.com/sectrean/di-kit"

var _ di.Scope = nil
