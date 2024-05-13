DI-Kit
======

**DI-Kit** is a dependency injection toolkit for modern Go applications.

## Usage

1. Create a new `Container` and register services.
2. `Resolve` services from the `Container`.
3. `Close` the `Container` when you're done.

```go
// Create the Container and register services using values and functions
c, err := di.NewContainer(
    di.RegisterValue(logger),
    di.RegisterFunc(foo.NewFooService),
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

## Register Services

Use `RegisterFunc` to register a constructor function for a service. The function may accept any number and type of arguments which must also be registered with the `Container`. The service will be registered with the function return type, and may also return an `error`.

Use `RegisterValue` to register a value with the container.

## Close Services

Services registered with `RegisterFunc` will automatically be closed if the resolved value implements any the following `Close` method signatures:

- `Close(context.Context) error`
- `Close(context.Context)`
- `Close() error`
- `Close()`

This behavior can be disabled using the `IgnoreCloser` option:

```go
c, err := di.NewContainer(
    di.RegisterValue(logger),
    di.RegisterFunc(foo.NewFooService, di.IgnoreCloser()),
)
```

If a service uses another method to clean up, a custom close function can be configured using the `WithCloseFunc` option:

``` go
c, err := di.NewContainer(
    di.RegisterValue(logger),
    di.RegisterFunc(foo.NewFooService,
        di.WithCloseFunc(func (ctx context.Context, fooSvc *foo.FooService) error {
            return fooSvc.Shutdown(ctx)
        }),
    ),
)
```

Services registered with `RegisterValue` will not be closed by default. Use the `WithCloser` option to call a supported `Close` method. Use the `WithCloseFunc` option to specify a custom close function. 

## Lifetimes



## Scopes

Create child scopes by creating a new `Container` and providing the parent `Container`.

```go
var requestVal MyRequestValue

childScope, err := di.NewContainer(
    di.WithParent(c),
    di.RegisterValue(requestVal)
)
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

Then the `di.Scope` can be retrieved from the context. 

```go
// Resolve from the `Scope` on the context.
svc, err := dicontext.Resolve[MyRequestValue](ctx)
```

