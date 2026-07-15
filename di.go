/*
Package di is a dependency injection toolkit for modern Go applications.

# Usage

The typical workflow has three steps:

 1. Create a [Container] with [NewContainer] and register services using values
    and constructor functions with the [WithService] option.

 2. Resolve services by type from the container using [Resolve], [MustResolve],
    or [Invoke].

 3. Close the container with [Container.Close] when you're done. The container
    calls Close on any services it created.

    // 1. Create the Container and register services.
    c, err := di.NewContainer(
    di.WithService(logger), // var logger *slog.Logger
    di.WithService(storage.NewDBStore, // NewDBStore(context.Context) (*storage.DBStore, error)
    di.As[storage.Store](),
    ),
    di.WithService(service.NewService), // NewService(*slog.Logger, storage.Store) *service.Service
    )
    // ...

    // 3. Close the Container when you're done.
    defer func() {
    err := c.Close(ctx)
    // ...
    }()

    // 2. Resolve services by type from the Container.
    svc, err := di.Resolve[*service.Service](ctx, c)

A service can be almost any named type (or pointer to a named type) including
structs, interfaces, functions, or basic types. Some types like error and
[context.Context] are reserved.

# Features

Services can be registered as interfaces they implement with [As], grouped and
injected as slices, and differentiated by tags with [WithTag] and [WithTagged].

Function services have configurable lifetimes—[Singleton] (the default),
[Scoped], and [Transient]—and are closed with the container by default. Closing
behavior is customizable with [IgnoreCloser], [UseCloser], and [UseCloseFunc].

Child scopes created with [Container.NewScope] isolate scoped services, and
[Module] lets you package reusable collections of registrations.

# Companion packages

  - The dicontext package stores a container scope on a [context.Context] so it
    can be used as a service locator.
  - The dihttp package provides net/http middleware that creates a new child
    scope for each request.

See the package README for detailed examples of each feature:
https://pkg.go.dev/github.com/sectrean/di-kit#section-readme
*/
package di
