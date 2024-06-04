di-kit
======

**di-kit** is a dependency injection toolkit for modern Go applications.
It's designed to be easy-to-use, unobtrusive, flexible, and performant.

## Usage

```go
// Create the Container and register services using values and constructor functions.
c, err := di.NewContainer(
    di.Register(logger),				// var logger *slog.Logger
    di.Register(myservice.NewService),	// NewService(*slog.Logger) *myservice.Service
)
// ...

// Close the Container when you're done
defer func() {
	err := c.Close(ctx)
	// ...
}()

// Resolve services by type from the Container
fooSvc, err := di.Resolve[*foo.FooService](ctx, c)
// ...

// Use your services
fooSvc.Run(ctx)
```

## Features

- Generics
- Lifetimes - Singleton, Scoped, and Transient
- Type aliases
- Support for interfaces
- Support for "closing" services
- Support for `context.Context` as a parameter
- Doesn't spread into your code
- Support for injecting a slice of services
- HTTP request scope middleware

## Registering Services

Use `Register` to register services with either a value or a constructor function.

The function may accept any number and type of arguments which must also be registered with the `Container`. The service will be registered as the function return type, and may also return an `error`.

## Closing Services

Services often need to do some clean up when they're done being used.
The `Container` can be responsible for closing services when the `Container` is closed.

Services that the `Container` *creates* will automatically be closed if it implements one the following `Close` method signatures:

- `Close(context.Context) error`
- `Close(context.Context)`
- `Close() error`
- `Close()`

This behavior can be disabled using the `IgnoreCloser` option:

```go
c, err := di.NewContainer(
    di.Register(logger),
    di.Register(foo.NewFooService, di.IgnoreCloser()),
)
```

If a service uses another method to clean up, a custom close function can be configured using the `WithCloseFunc` option:

``` go
c, err := di.NewContainer(
    di.Register(logger),
    di.Register(foo.NewFooService,
        di.WithCloseFunc(func (ctx context.Context, fooSvc *foo.FooService) error {
            return fooSvc.Shutdown(ctx)
        }),
    ),
)
```

Value services are not closed by default since they are not created by the Container. Use the `WithCloser` option to call a supported `Close` method. Use the `WithCloseFunc` option to specify a custom close function. 

## Lifetimes

DI-Kit supports three different lifetimes for registered services:

- **Singleton**: Only one instance of the service is created and reused every time it is resolved from the container. This is the default lifetime.
- **Scoped**: A new instance of the service is created for each child scope of the container.
- **Transient**: A new instance of the service is created every time it is resolved from the container.

Specify a lifetime when registering a function for a service:

```go
c, err := di.NewContainer(
	di.Register(NewRequestService, di.Scoped)
    di.Register(NewUserStore, di.Transient)
)
```

## Aliases

Use `As` to register a service as another type. This can be used to register a service as as an interface. The alias type must be assignable to the service type.

```go
c, err := di.NewContainer(
	di.Register(logger),
	di.Register(myservice.NewService,	// returns *myservice.Service
		di.As[myservice.Interface](),	// register as interface
		di.As[*myservice.Service](),	// register as actual type
	),
)
```

## Slices of Services

If you register multiple services of the same type, you can resolve a slice.

- Inject slice
- Variadic args
- Use for things like Healthchecks

## Scopes

Scopes are ...

Create child scopes for scoped by creating a new `Container` and providing the parent `Container`. Services can also be registered with the new scope.

```go
var scopeVal ScopeValue

scope, err := di.NewContainer(
    di.WithParent(c),
    di.Register(scopeVal)
)
// Close the scope when you're done
```

## Context

Use the `dicontext` package to attach `di.Scope` to a `context.Context`.

You can create child scopes and attach them to a context. Example HTTP middleware to create per-request scopes:

```go

```

Then the `di.Scope` can be retrieved from the context and used as a [service locator](https://en.wikipedia.org/wiki/Service_locator_pattern).

```go
// Resolve from the Container scope on the context.
svc, err := dicontext.Resolve[MyRequestValue](ctx)
```

# TODO

- [ ] Implement feature to inject `Future[T any]`
- [ ] Track child scopes to make sure all child scopes have been closed. Use closerMu.
- [ ] Support decorator functions `func(T) T`
- [ ] Implement additional Container options:
	- [ ] Validate dependencies--make sure all types are resolvable, no cycles
- [ ] Enable error stacktraces optionally
- [ ] Logging with `slog`
- [ ] Support for `Shutdown` functions like `Closer`?
- [ ] Support injecting dependencies of the same type with different tags?
