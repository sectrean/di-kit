DI-Kit
======

**DI-Kit** is a dependency injection toolkit for modern Go applications.

## Usage

1. Create a new `Container` and `Register` services.
2. `Resolve` services from the `Container`.
3. `Close` the `Container` when you're done.

```go
// Create the Container and register services using values and functions
c, err := di.NewContainer(
    di.Register(logger),
    di.Register(foo.NewFooService),
)
// ...handle error...

// Resolve services from the Container
fooSvc, err := di.Resolve[*foo.FooService](ctx, c)
// ...handle error...
fooSvc.Run(ctx)

// Close the Container when you're done
err = c.Close(ctx)
// ...handle error...
```

## Registering Services

Use `Register` to register a service with either a value or a constructor function.

The function may accept any number and type of arguments which must also be registered with the `Container`. The service will be registered as the function return type, and may also return an `error`.

## Close Services

The Container is responsible for closing services that it creates using value functions. Services will automatically be closed if the resolved value implements any the following `Close` method signatures:

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



## Scopes

Create child scopes by creating a new `Container` and providing the parent `Container`. Services can also be registered with the new scope.

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
func RequestScopeMiddleware(parent *di.Container) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new scope for the request
			scope, err := di.NewContainer(
				di.WithParent(parent),
				// TODO: Register any request-specific services here
			)
			if err != nil {
				// TODO: Log error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Add the scope to the request context
			ctx := dicontext.WithScope(r.Context(), scope)
			next.ServeHTTP(w, r.WithContext(ctx))

			// Close the scope when the request is done
			err = scope.Close(ctx)
			if err != nil {
				// TODO: Log error
			}
		})
	}
}
```

Then the `di.Scope` can be retrieved from the context and used as a [service locator](https://en.wikipedia.org/wiki/Service_locator_pattern).

```go
// Resolve from the `Scope` on the context.
svc, err := dicontext.Resolve[MyRequestValue](ctx)
```

# TODO

[ ] Support for injecting a `Future[T any]`: `WithFuture`
[ ] Decorators
[ ] Logging with `slog`
[ ] Enable error stacktraces optionally
[ ] Scope wrapper that will return errors until the service has finished resolving. Use channel for this?
[x] HTTP middleware for Scopes
[ ] Tasks/scripts for tests, benchmarking, codegen, etc. 
	https://taskfile.dev/

