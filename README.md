# 🧰 di-kit
[![Go Reference][go-img]][go]
[![Build Status][ci-img]][ci]
[![codecov][cov-img]][cov]

[go-img]: https://pkg.go.dev/badge/github.com/sectrean/di-kit.svg
[go]: https://pkg.go.dev/github.com/sectrean/di-kit
[ci-img]: https://github.com/sectrean/di-kit/actions/workflows/go.yaml/badge.svg
[ci]: https://github.com/sectrean/di-kit/actions/workflows/go.yaml
[cov-img]: https://codecov.io/gh/sectrean/di-kit/graph/badge.svg?token=EOSZDYMEEM
[cov]: https://codecov.io/gh/sectrean/di-kit

**di-kit** is a dependency injection toolkit for modern Go applications.
It's designed to be easy-to-use, lightweight, and full-featured.

## Usage

1. Create the `Container` and register *services* using values and constructor functions.
2. Resolve *services* by type from the `Container`.
3. Close the `Container` when you're done. The container will call `Close` on any services it created.

```go
// 1. Create the Container and register services using values and constructor functions.
c, err := di.NewContainer(
	di.WithService(logger),             // var logger *slog.Logger
	di.WithService(storage.NewDBStore, // NewDBStore(context.Context) (*storage.DBStore, error)
		di.As[storage.Store](),
	),
	di.WithService(service.NewService), // NewService(*slog.Logger, storage.Store) *service.Service
)
// ...

defer func() {
	// 3. Close the Container when you're done. 
	// The container will call Close on any services it created.
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
go get github.com/sectrean/di-kit
```
*Requires Go 1.22 or higher*

### Create the Container

Use `NewContainer` on application startup to create a `Container`. Register services using `di.WithService()` [functional options](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis) with a *value* or *constructor function*.

A *value* can be a struct or a pointer to a struct. When the value type is requested from the `Container`, this value will be returned. The service will be registered as the value's actual type, even if the variable is declared as an interface. A service registered with a value is referred to as a *value service*.

A *constructor function* may accept any parameters. The function must return a service, and may also return an error. When the service is requested from the `Container`, the function is called with the parameters resolved from the container. The service will be registered as the function's return type, which can be a struct, a pointer to a struct, or an interface. A service registered with a function is referred to as a *function service*.

```go
logger := slog.New(/*...*/)

c, err := di.NewContainer(
	// Value service registered as *slog.Logger 
	di.WithService(logger),
	// Function service registered as storage.Store
	di.WithService(storage.NewDBStore, di.As[storage.Store]()), // NewDBStore(context.Context) (*storage.DBStore, error)
	// Function service registered as *service.Service
	di.WithService(service.NewService), // NewService(*slog.Logger, storage.Store) *service.Service
)
```

Any errors from registering services will be [joined](https://pkg.go.dev/errors#Join) together.

### Resolve services

Use `Resolve` to get a service from the `Container` by type. If a requested service is not registered with the `Container`, or a dependency cycle is detected, an error will be returned.

```go
svc, _ := di.Resolve[*service.Service](ctx, c)
svc.Run(ctx)
```

Use `Invoke` to invoke a function using parameters resolved from the `Container`.

```go
// var c *di.Container
err = di.Invoke(ctx, c, runService)
```
```go
func runService(ctx context.Context, svc *service.Service) error {
	err := svc.Start(ctx)
	if err != nil {
		return err
	}

	// Wait ...

	return svc.Stop(ctx)	
}
```

### Close the Container

Services often need to do some clean up when they're done being used. The `Container` can handle this for registered services.

On application shutdown, use `Container.Close` to clean up services. By default, the `Container` will call a `Close` method on all services that is has created. Any errors returned from closing services will be [joined](https://pkg.go.dev/errors#Join) together.
See [Closing](#closing) for more.

## Features

### Interfaces

It's recommended that your service constructor functions *[accept interfaces and return structs](https://medium.com/@cep21/what-accept-interfaces-return-structs-means-in-go-2fe879e25ee8)*.

By default, function services are registered as the function return type.
Use the `di.As[Service]()` option to register a service as an interface that it implements. This allows your other services to depend on interfaces, which makes mocking/testing easier.

```go
c, err := di.NewContainer(
	di.WithService(storage.NewDBStore,	// NewDBStore() *storage.DBStore
		di.As[storage.Store](),	// Register the service as implemented interface
		di.As[*storage.DBStore](),	// Add this if you also want to use the function return type
	),
	di.WithService(service.NewService), // NewService(storage.Store) *service.Service
)
```

### Closing

By default, *function services* are closed with the `Container` if they implement one the following `Close` method signatures:

- `Close(context.Context) error`
- `Close(context.Context)`
- `Close() error`
- `Close()`

The default behavior can be disabled using the `di.IgnoreClose()` option when registering the service:

```go
c, err := di.NewContainer(
	di.WithService(logger),
	di.WithService(service.NewService,
		// We don't want the container to automatically call Close
		di.IgnoreClose(),
	),
)
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

*Value services* are not closed by default since they are not created by the `Container`. If you want to have the `Container` close a value service, use the `di.WithClose()` option to call a supported `Close` method. Or use the `di.WithCloseFunc()` option to specify a custom close function.

### Slice Services

If you register multiple services as the same type, you can inject all of them as a slice, or a variadic parameter.

```go
c, err := di.NewContainer(
	di.WithService(storage.NewDBStore, di.As[storage.Store](),
		di.As[healthcheck.HealthChecker](),
	),
	di.WithService(cache.NewRedisCache, di.As[cache.Cache](),
		di.As[healthcheck.HealthChecker](),
	),
	// All services registered as HealthChecker will be resolved and injected as a slice
	di.WithService(healthcheck.NewHealthHandler), // NewHealthHandler([]HealthChecker) *HealthHandler
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
	di.WithService(storage.NewReadWriteStore, // NewReadWriteStore(*db.DB) *storage.ReadWriteStore
		di.WithTagged[*db.DB](db.Primary),
	),
	di.WithService(storage.NewReadOnlyStore, // NewReadOnlyStore(*db.DB) *storage.ReadOnlyStore
		di.WithTagged[*db.DB](db.Replica),
	),
)
```

```go
// The *db.DB service tagged with db.Primary will be injected
rwStore, err := di.Resolve[*storage.ReadWriteStore](ctx, c)

// The *db.DB service tagged with db.Replica will be injected
roStore, err := di.Resolve[*storage.ReadOnlyStore](ctx, c)
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
	di.WithService(service.NewScopedService, di.ScopedLifetime),
	di.WithService(service.NewTransientService, di.TransientLifetime),
)
```

### Scopes

You can create new Containers with child scopes. Scoped dependencies can be resolved from a child scope. 

```go
c, err := di.NewContainer(
	di.WithService(logger),
	di.WithService(service.NewService),
	di.WithService(service.NewScopedService, di.ScopedLifetime),
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

`context.Context` - When a service is resolved, the context passed into `Resolve` will be injected into constructor functions as a dependency. You should avoid resolving resolving singleton services from a request-scoped context that may be canceled. 

`di.Scope` - The current `Container` can be injected into a service as `di.Scope`. This allows a service to resolve other services. The scope must be stored and only used *after* the constructor function returns.

```go
func NewDBFactory(scope di.Scope) *DBFactory {
	return &DBFactory{scope}
}

type DBFactory struct {
	scope di.Scope
}

func (f *DBFactory) NewDB(ctx context.Context, dbName string) *DB {
	// Use f.scope to resolve dependencies needed to create a *DB...
}
```

### Decorators

It's often useful to "wrap" or "decorate" a *service* to add some functionality.

Use `di.WithDecorator()` when creating a `Container` to register a decorator function.
A decorator function must accept and return a *service*. It may also accept other parameters which will be resolved from the container.

```go
c, err := di.NewContainer(
	di.WithService(logger), // var logger *slog.Logger
	di.WithService(service.NewService, // NewService() *service.Service
		di.As[service.Interface](),
	),
	di.WithDecorator(service.NewLoggedService), // NewLoggedService(service.Interface, *slog.Logger) service.Interface
)
// ...

svc, err := di.Resolve[service.Interface](ctx, c)
```

If you register multiple decorators for a service, they will be applied in the order they are registered.

### Modules

Modules allow you to export a collection of container options (services, decorators, etc.) that can be re-used for different containers.

```go
var DependencyModule = di.Module{
	di.WithService(NewLogger),
	//...
}
```

```go
c, err := di.NewContainer(
	di.WithModule(DependencyModule), // var DependencyModule di.Module
	di.WithService(NewService), // NewService(*slog.Logger) *service.Service
)
```

## `dicontext`

The `dicontext` package allows you to add a container scope to a `context.Context`.
Then the scope can be retrieved from the context and used as a [service locator](https://en.wikipedia.org/wiki/Service_locator_pattern).

```go
// Add container scope to the context
ctx = dicontext.WithScope(ctx, c)
```

```go
// Resolve services from the scope on the context
svc, err := dicontext.Resolve[*service.Service](ctx)
```

## `dihttp`

The `dihttp` package provides configurable `net/http` middleware to create new child scopes for each request. The scope is added to the request context using the `dicontext` package.

```go
c, err := di.NewContainer(
	di.WithService(logger),
	di.WithService(service.NewRequestService, di.ScopedLifetime), // NewRequestService(*slog.Logger, *http.Request) *service.RequestService
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

- Use `di.Lazy[Service any]` to inject a lazily-resolvable service.
	Can be used to avoid creation if service is never needed. Or to get around dependency cycles in a simpler way than injecting `di.Scope`.
- Add `dicontext.WithoutScope(context.Context)` to remove/hide a scope from child contexts. 
- Track child scopes to make sure all child scopes have been closed. 
	What do we do in this case? Close the child container(s)? Return an error? 
- Allow retrying `Resolve` if an error was returned. Normally the first error would be cached for singleton or scoped dependencies. Subsequent attempts to resolve the service will return the error. However, there may be some cases where you would want to be able to retry the constructor function.
- Implement additional Container options:
	- Validate services: make sure all types are resolvable, with no cycles.
		(Will need to exclude scoped services in the root container since they may have dependencies registered in child scopes.) 
- Automatically call `Shutdown` methods to close services.
- Enable error stacktraces optionally.
- Logging with `slog`.
