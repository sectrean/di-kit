package di

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// NewContainer creates a new Container with the provided options.
//
// Available options:
//   - [WithParent] specifies a parent Container.
//   - [RegisterFunc] registers a service with a constructor function.
//   - [RegisterValue] registers a service with a value.
func NewContainer(opts ...ContainerOption) (*Container, error) {
	c := &Container{
		services: make(map[serviceKey]service),
		resolved: make(map[serviceKey]resolvedService),
	}

	// Apply options
	var errs errors.MultiError
	for _, opt := range opts {
		err := opt.applyContainer(c)
		errs = errs.Append(err)
	}

	if err := errs.Join(); err != nil {
		return nil, errors.Wrap(err, "new container")
	}

	return c, nil
}

// Container allows you to resolve registered services.
type Container struct {
	parent    *Container
	services  map[serviceKey]service
	resolveMu sync.Mutex
	resolved  map[serviceKey]resolvedService
	closed    atomic.Bool
	closers   []Closer
}

var _ Scope = (*Container)(nil)

// Register registers the provided service.
func (c *Container) register(s service) {
	if len(s.Aliases()) == 0 {
		c.registerType(s.Type(), s)
	} else {
		for _, alias := range s.Aliases() {
			c.registerType(alias, s)
		}
	}
}

func (c *Container) registerType(t reflect.Type, s service) {
	key := serviceKey{Type: t}

	// Register a slice service if the type is already registered
	if existing, ok := c.services[key]; ok {
		sliceKey := serviceKey{
			Type: reflect.SliceOf(t),
		}

		var sliceSvc *sliceService
		if sliceSvc, ok = c.services[sliceKey].(*sliceService); !ok {
			// Create a new slice service with the existing service
			sliceSvc = newSliceService(t)
			sliceSvc.Add(existing)

			c.services[sliceKey] = sliceSvc
		}

		// Add the new service to the slice service
		sliceSvc.Add(s)
	}

	// Register the service with a tag
	if s.Tag() != nil {
		keyWithTag := serviceKey{
			Type: s.Type(),
			Tag:  s.Tag(),
		}
		c.services[keyWithTag] = s
	}

	// The last service registered for a type will win
	c.services[key] = s
}

// Contains returns true if the Container has a service registered for the given [reflect.Type].
//
// Available options:
//   - [WithTag] specifies a tag associated with the service.
func (c *Container) Contains(t reflect.Type, opts ...ContainsOption) bool {
	config := newContainsConfig(t, opts)
	key := config.serviceKey()

	return c.contains(key)
}

func (c *Container) contains(key serviceKey) bool {
	_, found := c.services[key]
	if !found && c.parent != nil {
		found = c.parent.contains(key)
	}
	return found
}

func (c *Container) root() *Container {
	if c.parent == nil {
		return c
	}
	return c.parent.root()
}

// Resolve returns a service for the given [reflect.Type].
//
// The type must be registered with the Container.
// This will return an error if the Container has been closed.
//
// Available options:
//   - [WithTag] specifies a tag associated with the service.
func (c *Container) Resolve(ctx context.Context, t reflect.Type, opts ...ResolveOption) (any, error) {
	config, err := newResolveConfig(t, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve %s", t)
	}

	key := config.serviceKey()

	// TODO: Benchmark concurrent Resolve calls and then see if we can optimize it.
	// We also need to think about a possible deadlocks like if a service injects a Scope and
	// then calls Resolve() in the constructor function.
	root := c.root()
	root.resolveMu.Lock()
	defer root.resolveMu.Unlock()

	if c.closed.Load() {
		return nil, errors.Wrapf(ErrContainerClosed, "resolve %s", t)
	}

	// Recursively resolve the type and its dependencies
	val, err := c.resolve(ctx, key, resolveVisitor{})
	return val, errors.Wrapf(err, "resolve %s", key)
}

func (c *Container) resolve(
	ctx context.Context,
	key serviceKey,
	visitor resolveVisitor,
) (any, error) {
	// Check if the type is a special type
	switch key.Type {
	case contextType:
		return ctx, nil
	case scopeType:
		return c, nil
	}

	// Look up the service
	// Look in ancestors if the service is not found
	var scope = c
	svc, ok := scope.services[key]
	if !ok {
		for scope.parent != nil {
			scope = scope.parent
			svc, ok = scope.services[key]
			if ok {
				break
			}
		}
	}

	if svc == nil {
		return nil, ErrTypeNotRegistered
	}

	// For scoped services, use the current container,
	// not the parent container that has the service
	if svc.Lifetime() == Scoped {
		scope = c
	}

	// Check if we've already resolved this service
	if rs, ok := scope.resolved[key]; ok {
		return rs.Val, rs.Err
	}

	// Throw an error if we've already visited this service
	if visited := visitor.Enter(key); visited {
		return nil, ErrDependencyCycle
	}
	defer visitor.Leave(key)

	// Recursively resolve dependencies
	var deps []any
	if len(svc.Dependencies()) > 0 {
		deps = make([]any, len(svc.Dependencies()))
		for i, depKey := range svc.Dependencies() {
			depVal, depErr := scope.resolve(ctx, depKey, visitor)
			if depErr != nil {
				// Stop at the first error
				return depVal, errors.Wrapf(depErr, "resolve dependency %s", depKey)
			}
			deps[i] = depVal
		}
	}

	// Check context for errors before creating the service
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create the instance and store the value and error
	val, err := svc.GetValue(deps)

	if svc.Lifetime() != Transient {
		scope.resolved[key] = resolvedService{val, err}
	}

	// Add Closer for the service
	if closer := svc.GetCloser(val); closer != nil {
		scope.closers = append(c.closers, closer)
	}

	return val, err
}

// Close closes the Container and all of its services.
//
// Services are closed in the reverse order they were resolved/created.
// Errors returned from closing services are joined together.
//
// Close will return an error if called more than once.
func (c *Container) Close(ctx context.Context) error {
	// Take resolve lock so no more services can be resolved
	c.resolveMu.Lock()
	defer c.resolveMu.Unlock()

	if c.closed.Swap(true) {
		return errors.Wrap(ErrContainerClosed, "already closed")
	}

	// TODO: Track child scopes to make sure all child scopes have been closed.

	// Close services in reverse order
	var errs errors.MultiError
	for i := len(c.closers) - 1; i >= 0; i-- {
		err := c.closers[i].Close(ctx)
		errs = errs.Append(err)
	}

	return errs.Wrap("close container")
}

type resolvedService struct {
	Val any
	Err error
}

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
