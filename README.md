🧰 di-kit
==========

**di-kit** is a dependency injection toolkit for modern Go applications.
It's designed to be easy-to-use, unobtrusive, flexible, and performant.

## Usage

1. Create the `Container` and register *services* using values and constructor functions.
2. Resolve *services* by type from the `Container`.
3. Close the `Container` when you're done.

```go
// 1. Create the Container and register services using values and constructor functions.
c, err := di.NewContainer(
	di.Register(logger),             // var logger *slog.Logger
	di.Register(storage.NewDBStore), // NewDBStore(context.Context) (storage.Store, error)
	di.Register(service.NewService), // NewService(*slog.Logger, storage.Store) *service.Service
)
// ...

defer func() {
	// 3. Close the Container when you're done.
	err := c.Close(ctx)
	// ...
}()

// 2. Resolve services by type from the Container.
svc, err := di.Resolve[*service.Service](ctx, c)
// ...
```

### Installation

```shell
go get github.com/johnrutherford/di-kit
```
*Requires Go 1.22+.*

### Registering Services

Use `di.Register()` to register services with either a value or a constructor function.

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

This behavior can be disabled using the `di.IgnoreCloser()` option:

```go
c, err := di.NewContainer(
	di.Register(logger),
	di.Register(service.NewService,
		// We don't want the container to automatically call Close
		di.IgnoreCloser(),
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
	di.Register(logger),
	di.Register(service.NewService,
		di.WithCloseFunc(func (ctx context.Context, svc *service.Service) error {
			return svc.Shutdown(ctx)
		}),
	),
)
```

Value services are not closed by default since they are not created by the Container. If you want to have the Container close the value service, use the `di.WithCloser()` option to call a supported `Close` method. Or use the `di.WithCloseFunc()` option to specify a custom close function.

## Features

### Aliases

Use the `di.As[T]()` option to register a service as the specified type.
This can be used to register a service as as an interface. The alias type must be assignable to the service type.

```go
c, err := di.NewContainer(
	// ...
	di.Register(service.NewService,	// returns *service.Service
		di.As[service.Interface](),	// register as interface
		di.As[*service.Service](),	// register as actual type
	),
)
```

### Keyed Services

Use `di.WithKey()` to differentiate between multiple services of the same type.
Use `di.WithKeyed[T]()` when registering a dependent service to specify the key for a dependency.

```go
c, err := di.NewContainer(
	di.Register(db.NewPrimaryDB, // NewPrimaryDB() (*db.DB, error)
		di.WithKey(db.Primary),
	),
	di.Register(db.NewReplicaDB, // NewReplicaDB() (*db.DB, error)
		di.WithKey(db.Replica),
	),
	di.Register(storage.NewReadWriteStore, // NewReadWriteStore(*db.DB) storage.*ReadWriteStore
		di.WithKeyed[db.DB](db.Primary),
	),
	di.Register(storage.NewReadOnlyStore, // NewReadOnlyStore(*db.DB) storage.*ReadOnlyStore
		di.WithKeyed[db.DB](db.Replica),
	),
)
```

Use `di.WithKey()` to specify a key when resolving a service directly from a container.

```go
primary, err := di.Resolve[db.DB](ctx, c, di.WithKey(db.Primary)) 
```

### Lifetimes

Lifetimes control how services are created:

- **Singleton**: Only one instance of the service is created and reused every time it is resolved from the container. This is the default lifetime.
- **Scoped**: A new instance of the service is created for each child scope of the container. See [Scopes](#scopes) for more information.
- **Transient**: A new instance of the service is created every time it is resolved from the container.

Specify a lifetime when registering a function for a service:

```go
c, err := di.NewContainer(
	di.Register(service.NewScopedService, di.Scoped),
	di.Register(service.NewTransientService, di.Transient),
)
```

### Slices of Services

If you register multiple services of the same type, you can resolve a slice.

- Inject slice
- Variadic args
- Use for things like Healthchecks

### Context

Use the `dicontext` package to attach a container to a `context.Context`.

```go
ctx = dicontext.WithScope(ctx, c)
```

Then the container can be retrieved from the context and used as a [service locator](https://en.wikipedia.org/wiki/Service_locator_pattern).

```go
// Resolve from the container on the context
svc, err := dicontext.Resolve[*service.Service](ctx)
```

### Scopes

Scopes are useful...

```go
c, err := di.NewContainer(
	di.Register(logger),
	di.Register(service.NewService),
	di.Register(service.NewScopedService, di.Scoped),
)
```

Create a new Container with a child scope:

```go
scope, err := c.NewScope()
//...

// Don't forget to Close the scope when you're done
defer func() {
	err := scope.Close(ctx)
	//...
}
```

### HTTP Request Scope Middleware

The `dihttp` package has configurable HTTP middleware to create a new child scope for each request.

```go
c, err := di.NewContainer(
	di.Register(logger),
	di.Register(service.NewRequestService, di.Scoped), // NewRequestService(*slog.Logger, *http.Request) *RequestService
)
// ...

scopeMiddleware := dihttp.NewScopeMiddleware(c)

var handler http.Handler
handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// Access the scope from the request context
	ctx := r.Context()
	scope := dicontext.Scope(ctx)
	svc, err := dicontext.Resolve[*service.RequestService](ctx)
	//...
})
handler = scopeMiddleware(handler)
```

# TODO

- [ ] Track child scopes to make sure all child scopes have been closed.
- [ ] Add support for "decorator" functions `func(T [, deps...]) T`
- [ ] Get around dependency cycles by injecting `di.Lazy[T any]`
- [ ] Implement additional Container options:
	- [ ] Validate dependencies--make sure all types are resolvable, no cycles?
- [ ] Support for `Shutdown` functions like `Closer`?
- [ ] Enable error stacktraces optionally?
- [ ] Logging with `slog`?
