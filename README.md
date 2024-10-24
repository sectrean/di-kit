# 🧰 di-kit 
[![Build Status][ci-img]][ci]

[ci-img]: https://github.com/johnrutherford/di-kit/actions/workflows/go.yaml/badge.svg
[ci]: https://github.com/johnrutherford/di-kit/actions/workflows/go.yaml

**di-kit** is a dependency injection toolkit for modern Go applications.
It's designed to be easy-to-use, lightweight, and full-featured.

## Usage

1. Create the `Container` and register *services* using values and constructor functions.
2. Resolve *services* by type from the `Container`.
3. Close the `Container` when you're done. The container will `Close` any services it created.

```go
// 1. Create the Container and register services using values and constructor functions.
c, err := di.NewContainer(
	di.WithService(logger),             // var logger *slog.Logger
	di.WithService(storage.NewDBStore), // NewDBStore(context.Context) (storage.Store, error)
	di.WithService(service.NewService), // NewService(*slog.Logger, storage.Store) *service.Service
)
// ...

defer func() {
	// 3. Close the Container when you're done. 
	// The container will Close any services it created.
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

Constructor functions may accept any number and type of parameters. These parameters will also be resolved from the `Container` and injected into the function. The function must return a service and may optionally return an error. The service is registered as the return type of the function. 

### Resolving Services

Services are resolved from the `Container` by calling `Resolve` with a type. Function services are resolved by first resolving parameters and then calling the registered function. The service (and error, optionally) will be returned by `Resolve`.

If a requested service is not registered with the `Container`, or a dependency cycle is detected, an error will be returned.

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

### Slice Services

If you register multiple services as the same type, you can inject all of them as a slice, or a variadic parameter.

```go
c, err := di.NewContainer(
	di.WithService(storage.NewDBStore, di.As[storage.Store](),
		di.As[Healthchecker](),
	),
	di.WithService(cache.NewRedisCache, di.As[cache.Cache](),
		di.As[Healthchecker](),
	),
	// All services registered as Healthchecker will be resolved and injected as a slice
	di.WithService(NewHealthcheckHandler), // NewHealthHandler([]Healtchecker) HealthHandler
)
```

### Tagged Services

If you want to register multiple services as the same type, but be able to differentiate them when resolving, use `di.WithTag()` when registering the service.

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

You can create new Containers with child scopes. Scoped dependencies can be resolved from a child scope. 

```go
c, err := di.NewContainer(
	di.WithService(logger),
	di.WithService(service.NewService),
	di.WithService(service.NewScopedService, di.Scoped),
)

scope, err := c.NewScope()
// ...

// Don't forget to Close the scope when you're done
defer func() {
	err := scope.Close(ctx)
	// ...
}
```

New services can also be registered when creating a child scope. These new services are isolated from the parent or sibling Containers.

```go
scope, err := c.NewScope(
	di.WithService(requestService),
)
```

### Special Services

A couple services are provided directly by the container and cannot be registered.

`context.Context` - When a service is resolved, the context passed into `Resolve` will be injected into constructor functions as a dependency. You should avoid resolving resolving Singleton services from a request-scoped context that may be canceled. 

`di.Scope` - The current `Container` can be injected into a service as `di.Scope`. This allows a service to resolve other services. The scope must be stored and only used *after* the constructor function returns.

```go
func NewWidgetFactory(scope di.Scope) *WidgetFactory {
	return &WidgetFactory{scope}
}

type WidgetFactory struct {
	scope di.Scope
}

func (f *WidgetFactory) BuildWidget(color, shape string) *Widget {
	// Use f.scope to resolve dependencies need to create *Widget
	// ...
}
```

### Decorators

It's often useful to "wrap" or "decorate" a *service* to add some functionality.

Use `di.WithDecorator()` when creating a `Container` to register a decorator function.
A decorator function must accept and return a *service*. It may also accept other parameters which will be resolved from the container.

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
	di.WithService(service.NewRequestService, di.Scoped), // NewRequestService(*slog.Logger, *http.Request) *service.RequestService
)
// ...

var handler http.Handler
handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// Access the scope from the request context
	ctx := r.Context()
	svc, err := dicontext.Resolve[*service.RequestService](ctx)
	// ...
})

// Create and apply the middleware
scopeMiddleware := dihttp.RequestScopeMiddleware(c)
handler = scopeMiddleware(handler)
// ...
```

## Feature Ideas

- Implement `di.Lazy[Service any]` to inject a lazily-resolvable service.
	Can be used to avoid creation if service is never needed. Or to get around dependency cycles in a simpler way than injecting `di.Scope`.
- Track child scopes to make sure all child scopes have been closed. 
	What do we do in this case? Close the child container(s)? Return an error? 
- Allow retrying `Resolve` if an error was returned. Normally the first error would be cached for singleton or scoped dependencies. Subsequent attempts to resolve the service will return the error. However, there may be some cases where you would want to be able to retry the constructor function.
- Implement additional Container options:
	- Validate services: make sure all types are resolvable, with no cycles.
		(Will need to exclude scoped services in the root container since they may have dependencies registered in child scopes.) 
- Automatically call `Shutdown` methods to close services.
- Enable error stacktraces optionally.
- Logging with `slog`.
