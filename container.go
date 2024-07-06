package di

import (
	"cmp"
	"context"
	"maps"
	"reflect"
	"slices"
	"sync"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// Container is a dependency injection container.
// It is used to resolve services by first resolving their dependencies.
type Container struct {
	parent *Container

	services   map[serviceKey]service
	decorators map[serviceKey][]*decorator

	resolvedMu sync.Mutex
	resolved   map[serviceKey]resolvedService

	closersMu sync.Mutex
	closers   []Closer

	closedMu sync.RWMutex
	closed   bool
}

var _ Scope = (*Container)(nil)

// NewContainer creates a new Container with the provided options.
//
// Available options:
//   - [WithService] registers a service with a value or a function.
func NewContainer(opts ...ContainerOption) (*Container, error) {
	c := &Container{
		services: make(map[serviceKey]service),
		resolved: make(map[serviceKey]resolvedService),
	}

	// Sort options by precedence
	slices.SortStableFunc(opts, func(a, b ContainerOption) int {
		return cmp.Compare(a.order(), b.order())
	})

	// Apply options
	var errs errors.MultiError
	for _, opt := range opts {
		err := opt.applyContainer(c)
		errs = errs.Append(err)
	}

	if len(errs) > 0 {
		return nil, errs.Wrap("new container")
	}

	return c, nil
}

func (c *Container) register(sr serviceRegistration) {
	// Child containers point to the same services map as the parent container initially.
	// If we're registering new services in the child container,
	// we need to clone the parent map first.
	if c.parent != nil && &c.parent.services == &c.services {
		c.services = maps.Clone(c.parent.services)
	}

	if len(sr.Aliases()) == 0 {
		c.registerType(sr.Type(), sr)
	} else {
		for _, alias := range sr.Aliases() {
			c.registerType(alias, sr)
		}
	}

	// Pre-resolve value services and add closer
	// We don't need to take locks here because this is only called when creating a new Container
	if vs, ok := sr.(*valueService); ok {
		c.resolved[sr.Key()] = valueResult{vs.val}

		if closer := sr.AsCloser(vs.val); closer != nil {
			c.closers = append(c.closers, closer)
		}
	}
}

func (c *Container) registerType(t reflect.Type, sr serviceRegistration) {
	// The last service registered for a type will win
	key := serviceKey{Type: t}
	c.services[key] = sr

	// Register the service with a tag if it has one
	if sr.Tag() != nil {
		keyWithTag := serviceKey{
			Type: t,
			Tag:  sr.Tag(),
		}
		c.services[keyWithTag] = sr
	}

	// Add the service to a slice service
	sliceKey := serviceKey{Type: reflect.SliceOf(t)}
	sliceSvc, ok := c.services[sliceKey].(*sliceService)
	if !ok {
		sliceSvc = newSliceService(t)
		c.services[sliceKey] = sliceSvc
	}

	itemKey := sliceSvc.NextItemKey()
	c.services[itemKey] = sr
}

func (c *Container) registerDecorator(d *decorator) {
	// TODO: Validate that the service being decorated is registered?

	// Create this map lazily since decorators aren't always used
	if c.decorators == nil {
		c.decorators = make(map[serviceKey][]*decorator)
	}

	decorators := c.decorators[d.Key()]
	decorators = append(decorators, d)
	c.decorators[d.Key()] = decorators
}

// NewScope creates a new Container with a child scope.
//
// Services registered with the parent Container will be inherited by the child Container.
// Additional services can be registered with the new Scope if needed.
// They will only be available to the new scope.
//
// Available options:
//   - [WithService] registers a service with a value or a function.
func (c *Container) NewScope(opts ...ContainerOption) (*Container, error) {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()

	if c.closed {
		return nil, errors.Wrap(ErrContainerClosed, "new scope")
	}

	scope := &Container{
		parent:   c,
		services: c.services,
		resolved: make(map[serviceKey]resolvedService),
	}

	// Apply options
	var errs errors.MultiError
	for _, opt := range opts {
		err := opt.applyContainer(scope)
		errs = errs.Append(err)
	}

	if len(errs) > 0 {
		return nil, errs.Wrap("new scope")
	}

	return scope, nil
}

// Contains returns true if the Container has a service registered for the given [reflect.Type].
//
// Available options:
//   - [WithTag] specifies a key associated with the service.
func (c *Container) Contains(t reflect.Type, opts ...ResolveOption) bool {
	key := serviceKey{Type: t}
	for _, opt := range opts {
		key = opt.applyServiceKey(key)
	}

	return c.contains(key)
}

func (c *Container) contains(key serviceKey) bool {
	_, found := c.services[key]
	return found
}

func (c *Container) root() *Container {
	if c.parent == nil {
		return c
	}
	return c.parent.root()
}

// ResolveOption can be used when calling [Resolve], [MustResolve],
// [Container.Resolve], or [Container.Contains].
//
// Available options:
//   - [WithTag]
type ResolveOption interface {
	applyServiceKey(serviceKey) serviceKey
}

// Resolve a service of the given [reflect.Type].
//
// The type must be registered with the Container.
// This will return an error if the Container has been closed.
//
// Available options:
//   - [WithTag] specifies a key associated with the service.
func (c *Container) Resolve(ctx context.Context, t reflect.Type, opts ...ResolveOption) (any, error) {
	key := serviceKey{Type: t}
	for _, opt := range opts {
		key = opt.applyServiceKey(key)
	}

	c.closedMu.RLock()
	defer c.closedMu.RUnlock()

	if c.closed {
		return nil, errors.Wrapf(ErrContainerClosed, "resolve %s", key)
	}

	val, err := c.resolve(ctx, key, make(resolveVisitor))
	return val, errors.Wrapf(err, "resolve %s", key)
}

func (c *Container) resolve(
	ctx context.Context,
	key serviceKey,
	visitor resolveVisitor,
) (val any, err error) {
	// Look up the service
	svc, registered := c.services[key]
	if !registered {
		return nil, ErrServiceNotRegistered
	}

	// Throw an error if we've already visited this service
	if visited := visitor.Enter(key); visited {
		return nil, ErrDependencyCycle
	}
	defer visitor.Leave(key)

	// For scoped services, use the current container.
	// For singleton services, use the root container.
	// TODO: We actually need to use the scope that the service was registered with
	scope := c
	if svc.Lifetime() == Singleton {
		scope = c.root()
	}

	// For Singleton or Scoped services, we store a promise for each service.
	// The first request for a service will create the promise and then
	// continue to resolve the service and set the result.
	// Subsequent requests will just wait on the promise.
	if svc.Lifetime() != Transient {
		scope.resolvedMu.Lock()

		res, exists := scope.resolved[svc.Key()]
		if !exists {
			// Create a promise that will be resolved when this function returns
			promise, resolvePromise := newServicePromise()
			scope.resolved[svc.Key()] = promise
			res = promise

			defer func() {
				resolvePromise(val, err)
			}()
		}

		scope.resolvedMu.Unlock()

		if exists {
			// This will block until the value and error are set
			// by the first request to resolve this service.
			return res.Result()
		}
	}

	// Recursively resolve dependencies
	var deps []reflect.Value
	if len(svc.Dependencies()) > 0 {
		deps = make([]reflect.Value, len(svc.Dependencies()))
		for i, depKey := range svc.Dependencies() {
			var depVal any
			var depErr error

			switch depKey.Type {
			case contextType:
				// Pass along the context
				depVal = ctx

			case scopeType:
				// We wrap the scope to prevent Resolve from being called on it
				// until we finish resolving this service. Otherwise it could
				// cause a deadlock.
				//
				// TODO: There may still be a way to deadlock. Write some tests around this.
				var ready func()
				depVal, ready = newInjectedScope(scope, key)
				defer ready()

			default:
				// Recursive call
				depVal, depErr = scope.resolve(ctx, depKey, visitor)
			}

			if depErr != nil {
				// Stop at the first error
				return nil, errors.Wrapf(depErr, "dependency %s", depKey)
			}
			deps[i] = safeVal(depKey.Type, depVal)
		}
	}

	// Check context for errors before creating the service
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create the service
	val, err = svc.New(deps)
	if err != nil {
		return val, err
	}

	// Apply decorators
	if decorators, exists := c.decorators[key]; exists {
		for _, d := range decorators {
			// Resolve dependencies for the decorator
			deps := make([]reflect.Value, len(d.deps))
			for i, depKey := range d.deps {
				var depVal any
				var depErr error

				// TODO: Support di.Scope as a decorator dependency?
				switch {
				case depKey == key:
					// Inject the service being decorated
					depVal = val

				case depKey.Type == contextType:
					// Pass along the context
					depVal = ctx

				default:
					// Recursive call
					depVal, depErr = scope.resolve(ctx, depKey, visitor)
				}

				if depErr != nil {
					return nil, errors.Wrapf(depErr, "decorator %s: dependency %s", d, depKey)
				}
				deps[i] = safeVal(depKey.Type, depVal)
			}

			// Apply the decorator
			val = d.Decorate(deps)
		}
	}

	// Add Closer for the service
	if closer := svc.AsCloser(val); closer != nil {
		scope.closersMu.Lock()
		scope.closers = append(scope.closers, closer)
		scope.closersMu.Unlock()
	}

	return val, nil
}

// Close the Container and resolved services.
//
// Services are closed in the reverse order they were resolved/created.
// Errors returned from closing services are joined together.
//
// Close will return an error if called more than once.
func (c *Container) Close(ctx context.Context) error {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()

	if c.closed {
		return errors.Wrap(ErrContainerClosed, "close: already closed")
	}
	c.closed = true

	// Close services in reverse order
	var errs errors.MultiError
	for i := len(c.closers) - 1; i >= 0; i-- {
		err := c.closers[i].Close(ctx)
		errs = errs.Append(err)
	}

	return errs.Wrap("close")
}

var (
	// ErrServiceNotRegistered is returned when a service is not registered.
	ErrServiceNotRegistered = errors.New("service not registered")

	// ErrDependencyCycle is returned when a dependency cycle is detected.
	ErrDependencyCycle = errors.New("dependency cycle detected")

	// ErrContainerClosed is returned when the container is closed.
	ErrContainerClosed = errors.New("container closed")

	// Common types

	errorType   = reflect.TypeFor[error]()
	contextType = reflect.TypeFor[context.Context]()
	scopeType   = reflect.TypeFor[Scope]()
)

type resolveVisitor map[serviceKey]struct{}

func (v resolveVisitor) Enter(key serviceKey) (visited bool) {
	if _, exists := v[key]; exists {
		return true
	}

	v[key] = struct{}{}
	return false
}

func (v resolveVisitor) Leave(key serviceKey) {
	delete(v, key)
}
