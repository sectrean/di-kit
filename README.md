# 🧰 di-kit 
[![Build Status][ci-img]][ci]

[ci-img]: https://github.com/johnrutherford/di-kit/actions/workflows/go.yaml/badge.svg
[ci]: https://github.com/johnrutherford/di-kit/actions/workflows/go.yaml

**di-kit** is a dependency injection toolkit for modern Go applications.
It's designed to be easy-to-use, lightweight, and full-featured.

## Usage

1. Create the `Container` and register your *services* using values and constructor functions.
2. Resolve *services* by type from the `Container`.
3. Close the `Container` and *services* when you're done.

```go
// 1. Create the Container and register services using values and constructor functions.
c, err := di.NewContainer(
	di.WithService(logger),             // var logger *slog.Logger
	di.WithService(storage.NewDBStore), // NewDBStore(context.Context) (storage.Store, error)
	di.WithService(service.NewService), // NewService(*slog.Logger, storage.Store) *service.Service
)
// ...

// 3. Close the Container and services when you're done.
defer func() {
	err := c.Close(ctx)
	// ...
}()

// 2. Resolve services by type from the Container.
svc, err := di.Resolve[*service.Service](ctx, c)
// ...
```

## Getting Started

### Install

```shell
go get github.com/johnrutherford/di-kit
```
*Requires Go 1.22+.*

### Registering Services

Use `di.WithService()` to register services with either a value or a constructor function.

The function may accept any number and type of arguments which must also be registered with the `Container`. The service will be registered as the function return type, and may also return an `error`.

### Resolving Services



### Closing Services

Services often need to do some clean up when they're done being used.
The `Container` can be responsible for closing services when the `Container` is closed.

By default, services that the `Container` *creates* (registered with a function, not value) will automatically be closed if they implement one the following `Close` method signatures:

- `Close(context.Context) error`
- `Close(context.Context)`
- `Close() error`
- `Close()`

This behavior can be disabled using the `di.IgnoreClose()` option:

```go
c, err := di.NewContainer(
	di.WithService(logger),
	di.WithService(service.NewService,
		// We don't want the container to automatically call Close
		di.IgnoreClose(),
	),
)
// ...

svc := di.MustResolve[*service.Service](ctx, c)
// We want to close it manually
defer svc.Close(ctx)
```

If a service uses another method to clean up, a custom close function can be configured using the `di.WithCloseFunc()` option:

``` go
c, err := di.NewContainer(
	di.WithService(logger),
	di.WithService(service.NewService,
		di.WithCloseFunc(func (ctx context.Context, svc *service.Service) error {
			return svc.Shutdown(ctx)
		}),
	),
)
```

Value services are not closed by default since they are not created by the Container. If you want to have the Container close the value service, use the `di.WithClose()` option to call a supported `Close` method. Or use the `di.WithCloseFunc()` option to specify a custom close function.

## Features

### Aliases

Use the `di.As[Service]()` option to register a service as the specified type.
This can be used to register a service as as an interface. The alias type must be assignable to the service type.

```go
c, err := di.NewContainer(
	// ...
	di.WithService(service.NewService,	// returns *service.Service
		di.As[service.Interface](),	// register as interface
		di.As[*service.Service](),	// also register as actual type
	),
)
```

### Tagged Services

Use `di.WithTag()` when registering a service to differentiate between different services of the same type.

Use `di.WithTagged[Dependency]()` when registering a dependent service to specify a tag for a dependency.

```go
c, err := di.NewContainer(
	di.WithService(db.NewPrimaryDB, // NewPrimaryDB(context.Context) (*db.DB, error)
		di.WithTag(db.Primary),
	),
	di.WithService(db.NewReplicaDB, // NewReplicaDB(context.Context) (*db.DB, error)
		di.WithTag(db.Replica),
	),
	di.WithService(storage.NewReadWriteStore, // NewReadWriteStore(*db.DB) storage.*ReadWriteStore
		di.WithTagged[*db.DB](db.Primary),
	),
	di.WithService(storage.NewReadOnlyStore, // NewReadOnlyStore(*db.DB) storage.*ReadOnlyStore
		di.WithTagged[*db.DB](db.Replica),
	),
)
```

Use `di.WithTag()` to specify a tag when resolving a service directly from a container.

```go
primary, err := di.Resolve[*db.DB](ctx, c, di.WithTag(db.Primary)) 
```

### Slice Services

If you register multiple services of the same type, you can resolve a slice.

- Inject slice
- Variadic args
- Use for things like Healthchecks

### Lifetimes

Lifetimes control how function services are created:

- `Singleton`: Only one instance of the service is created and reused every time it is resolved from the container. This is the default lifetime.
- `Scoped`: A new instance of the service is created for each child scope of the container. See [Scopes](#scopes) for more information.
- `Transient`: A new instance of the service is created every time it is resolved from the container.

Specify a lifetime when registering a function service:

```go
c, err := di.NewContainer(
	di.WithService(service.NewScopedService, di.Scoped),
	di.WithService(service.NewTransientService, di.Transient),
)
```

### Scopes

Scopes are useful...

```go
c, err := di.NewContainer(
	di.WithService(logger),
	di.WithService(service.NewService),
	di.WithService(service.NewScopedService, di.Scoped),
)
```

Create a new Container with a child scope:

```go
scope, err := c.NewScope()
// ...

// Don't forget to Close the scope when you're done
defer func() {
	err := scope.Close(ctx)
	// ...
}
```

### Decorators

It's often useful to "wrap" or "decorate" a *service* to add some functionality.

Use `di.WithDecorator()` when creating a `Container` to register a decorator function.
A decorator function must accept and return a *service*. It may also accept other arguments which will be resolved from the container.

```go
c, err := di.NewContainer(
	di.WithService(logger), // var logger *slog.Logger
	di.WithService(service.NewService), // NewService() service.Interface
	di.WithDecorator(service.NewLoggedService), // NewLoggedService(service.Interface, *slog.Logger) service.Interface
)
// ...

svc, err := di.Resolve[service.Interface](ctx, c)
```

If you register multiple decorators for a service, they will be applied in the order they are registered. Value services cannot be decorated.

## `dicontext`

This package allows you to add a container scope to a `context.Context`.
Then the scope can be retrieved from the context and used as a [service locator](https://en.wikipedia.org/wiki/Service_locator_pattern).

```go
// Add container scope to context
ctx = dicontext.WithScope(ctx, c)
```

```go
// Get container scope from the context
scope := dicontext.Scope(ctx)
```

```go
// Resolve from the scope on the context
svc, err := dicontext.Resolve[*service.Service](ctx)
```

## `dihttp`

The `dihttp` package provides configurable HTTP middleware to create new child scopes for each request.

```go
c, err := di.NewContainer(
	di.WithService(logger),
	// NewRequestService(*slog.Logger, *http.Request) *service.RequestService
	di.WithService(service.NewRequestService, di.Scoped),
)
// ...

scopeMiddleware := dihttp.RequestScopeMiddleware(c)

var handler http.Handler
handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// Access the scope from the request context
	ctx := r.Context()
	svc, err := dicontext.Resolve[*service.RequestService](ctx)
	// ...
})
handler = scopeMiddleware(handler)
// ...
```

## TODO

- Get around dependency cycles by injecting `di.Lazy[Service any]`
- Track child scopes to make sure all child scopes have been closed. 
	What do we do in this case? Close the child container(s)? Return an error? 
- Implement additional Container options:
	- Validate services: make sure all types are resolvable, with no cycles.
		(Will need to exclude scoped services in the root container since they may have dependencies registered in child scopes.) 

- Support for `Shutdown` functions like `Closer`?
- Enable error stacktraces optionally?
- Logging with `slog`?
