# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Documentation

- Documented the "last registration wins" behavior for single-value resolution when
  multiple services are registered for the same type and tag (including no tag). Resolve
  the type as a slice to get all of them, or use `di.WithTag()` with a distinct tag to
  select a specific one. No behavior change.

### Added

- `dihttp.WithRequestService` option to register the current `*http.Request` with each
  request scope, so it can be resolved directly or injected into scoped services.

### Changed

- **BREAKING:** `dihttp.NewRequestScopeMiddleware` no longer registers the current
  `*http.Request` with the request scope by default. Registration is now opt-in via the
  new `dihttp.WithRequestService` option. Scoped services that depend on `*http.Request`
  will fail to resolve unless the middleware is created with this option.
- Reduced per-request allocations when creating a scope. Child scopes now store their
  registrations in a slice instead of allocating a map, and the resolution cache is
  allocated lazily on first use. A request scope that registers no services dropped from
  9 to 3 allocations.

[Unreleased]: https://github.com/sectrean/di-kit/compare/v0.25.0...HEAD
