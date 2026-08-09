# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- The minimum supported Go version is now 1.23. Now using iterators.

### Fixed

- The `dihttp` scope middleware now correctly defers the closing of the request-scoped container.
  This will ensure that the scope still gets closed if a downstream handler panics.
- `di.ValidateContainer` now validates every registration that can satisfy a slice dependency,
  including registrations inherited by child scopes, instead of checking only a single service.
- `di.ValidateContainer` now reports all dependency validation failures in a stable order,
  de-duplicates services registered under multiple aliases, and validates scoped services inherited
  from all ancestor scopes.
- Variadic dependencies are now treated as optional only for the final variadic parameter, and any
  registered services for that variadic type are still validated.
- `Container.Close` now returns an error while the container is still in use by
  an open child scope or an in-progress resolution, this enforces consistency for service and container lifetimes.
- `di.Invoke` now returns an error when called with a nil function instead of
  panicking.
- `di.Invoke` now treats variadic dependencies as optional only when no matching
  service is registered, preserving resolution errors for registered services.
- `di.WithTag` and `di.WithTagged` options now reject non-comparable tags with an
  error instead of panicking when the tag is used as part of a service key.

### Changed

- Fixed some typos and formatting in docs.
- Update error message formatting to be more consistent. Errors from generic options like `di.As` and 
  `di.WithTagged` will include the type parameter in brackets.

## [0.26.0] - 2026-07-28

### Fixed

- A function service that declares an `error` parameter is now rejected at registration
  with an "invalid dependency type" error. Previously it registered successfully but
  always failed to resolve, since `error` cannot be provided as a dependency.

### Documentation

- Documented the "last registration wins" behavior for single-value resolution when
  multiple services are registered for the same type and tag (including no tag). Resolve
  the type as a slice to get all of them, or use `di.WithTag()` with a distinct tag to
  select a specific one. No behavior change.
- Documented on `di.WithService` which types cannot be registered as services and which
  types are valid as function-service dependencies.
- Added a more complete overview to the package-level godoc.
- Added Features -> Validation section to README.

### Added

- `dihttp.WithRequestService` option to register the current `*http.Request` with each
  request scope, so it can be resolved directly or injected into scoped services.
- `Contains[Service]` generic function for checking if a service is registered with a `Scope`.

### Changed

- **BREAKING:** `dihttp.NewRequestScopeMiddleware` no longer registers the current
  `*http.Request` with the request scope by default. Registration is now opt-in via the
  new `dihttp.WithRequestService` option. Scoped services that depend on `*http.Request`
  will fail to resolve unless the middleware is created with this option.
- Reduced per-request allocations when creating a scope. Child scopes now store their
  registrations in a slice instead of allocating a map, and the resolution cache is
  allocated lazily on first use. A request scope that registers no services dropped from
  9 to 3 allocations.
- **BREAKING** Refactored validation API to use a package-level `di.ValidateContainer()`
  function, rather than a `di.WithDependencyValidation()` option that returns an error when
  creating a `Container`. This change simplifies the API and makes it more flexible. A user
  can pass in a pre-constructed `Container` to validate rather than forcing them to append
  a container option.

### Removed

**BREAKING:** Removed the `WithModule` option. It was redundant
because a `Module` can be used directly as a `ContainerOption`.
**BREAKING:** Removed the `ditest` package. Two assert functions didn't justify the need
for this sub-package. Use `di.Contains[Service]` and assert on the result.

[Unreleased]: https://github.com/sectrean/di-kit/compare/v0.26.0...HEAD
[0.26.0]: https://github.com/sectrean/di-kit/compare/v0.25.0...v0.26.0
