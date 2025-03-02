package di

import (
	"cmp"
	"context"
	"reflect"
	"slices"
	"sync"

	"github.com/sectrean/di-kit/internal/errors"
)

// Container is a dependency injection container.
// It is used to resolve services by first resolving their dependencies.
type Container struct {
	parent     *Container
	services   map[serviceKey]service
	decorators map[serviceKey][]*decorator
	resolved   map[serviceKey]resolveResult
	closers    []Closer
	resolvedMu sync.RWMutex
	closedMu   sync.RWMutex
	closersMu  sync.Mutex
	closed     bool
}

var _ Scope = (*Container)(nil)

// NewContainer creates a new [Container] with the provided options.
//
// Available options:
//   - [WithService] registers a service with a value or constructor function.
//   - [WithDecorator] registers a decorator function.
func NewContainer(opts ...ContainerOption) (*Container, error) {
	c := &Container{
		// Pre-allocate space for services. This will not be accurate if modules or decorators are
		// used, but it's a probably better than the default starting size.
		services: make(map[serviceKey]service, len(opts)),
		resolved: make(map[serviceKey]resolveResult),
	}

	err := c.applyOptions(opts)
	if err != nil {
		return nil, errors.Wrap(err, "di.NewContainer")
	}

	return c, nil
}

// ContainerOption is used to configure a new [Container] when calling [NewContainer]
// or [Container.NewScope].
type ContainerOption interface {
	order() optionOrder
	applyContainer(*Container) error
}

func (c *Container) applyOptions(opts []ContainerOption) error {
	// Flatten any modules before sorting and applying options
	for i := range opts {
		if mod, ok := opts[i].(Module); ok {
			opts = slices.Insert(opts, i+1, mod...)
		}
	}

	// Sort options by precedence
	// Use stable sort because the registration order of services matters
	slices.SortStableFunc(opts, func(a, b ContainerOption) int {
		return cmp.Compare(a.order(), b.order())
	})

	var errs []error
	for _, o := range opts {
		err := o.applyContainer(c)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (c *Container) register(sc serviceConfig) {
	if len(sc.Assignables()) == 0 {
		c.registerType(sc.Type(), sc)
	} else {
		for _, assignable := range sc.Assignables() {
			c.registerType(assignable, sc)
		}
	}

	// Pre-resolve value services and add closer
	// We don't need to take locks here because this is only called when creating a new Container
	if vs, ok := sc.(*valueService); ok {
		c.resolved[sc.Key()] = resolveResult{val: vs.val}

		if closer := sc.CloserFor(vs.val); closer != nil {
			c.closers = append(c.closers, closer)
		}
	}
}

func (c *Container) registerType(t reflect.Type, sc serviceConfig) {
	// The last service registered for a type will win
	key := serviceKey{Type: t}
	c.services[key] = sc

	// Register the service with a tag if it has one
	if sc.Tag() != nil {
		keyWithTag := serviceKey{
			Type: t,
			Tag:  sc.Tag(),
		}
		c.services[keyWithTag] = sc
	}

	// Add the service to a slice service
	sliceKey := serviceKey{Type: reflect.SliceOf(t)}
	sliceSvc, ok := c.services[sliceKey].(*sliceService)
	if !ok {
		sliceSvc = newSliceService(t)
		c.services[sliceKey] = sliceSvc
	}

	itemKey := sliceSvc.NextItemKey()
	c.services[itemKey] = sc
}

func (c *Container) registerDecorator(d *decorator) {
	// Create this map lazily since decorators aren't always used
	if c.decorators == nil {
		c.decorators = make(map[serviceKey][]*decorator)
	}

	c.decorators[d.Key()] = append(c.decorators[d.Key()], d)
}

// NewScope creates a new [Container] with a child scope.
//
// Services registered with the parent [Container] will be inherited by the child [Container].
// Additional services can be registered with the new scope if needed and they will be isolated from
// the parent and sibling containers.
//
// Available options:
//   - [WithService] registers a service with a value or a function.
func (c *Container) NewScope(opts ...ContainerOption) (*Container, error) {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()

	if c.closed {
		return nil, errors.Wrap(ErrContainerClosed, "di.Container.NewScope")
	}

	scope := &Container{
		parent:   c,
		resolved: make(map[serviceKey]resolveResult),
	}

	if len(opts) > 0 {
		scope.services = make(map[serviceKey]service, len(opts))

		err := scope.applyOptions(opts)
		if err != nil {
			return nil, errors.Wrap(err, "di.Container.NewScope")
		}
	}

	return scope, nil
}

// Contains returns true if the [Container] has a service registered for the given [reflect.Type].
//
// Available options:
//   - [WithTag] specifies a key associated with the service.
func (c *Container) Contains(t reflect.Type, opts ...ResolveOption) bool {
	key := serviceKey{Type: t}
	for _, opt := range opts {
		key = opt.applyServiceKey(key)
	}

	_, _, registered := getService(c, key)
	return registered
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
// The type must be registered with the [Container].
// This will return an error if the [Container] has been closed.
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
		return nil, errors.Wrapf(ErrContainerClosed, "di.Container.Resolve %s", key)
	}

	val, err := resolve(ctx, c, key, make(resolveVisitor))
	if err != nil {
		return val, errors.Wrapf(err, "di.Container.Resolve %s", key)
	}

	return val, nil
}

func getService(scope *Container, key serviceKey) (service, *Container, bool) {
	for ; scope != nil; scope = scope.parent {
		svc, ok := scope.services[key]
		if ok {
			return svc, scope, true
		}
	}

	return nil, nil, false
}

func resolve(
	ctx context.Context,
	scope *Container,
	key serviceKey,
	visitor resolveVisitor,
) (val any, err error) {
	// Look up the service
	svc, svcScope, registered := getService(scope, key)
	if !registered {
		return nil, ErrServiceNotRegistered
	}

	// Check context for errors
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// For singleton services, use the scope the service is registered with.
	// Otherwise, use the current scope.
	lifetime := svc.Lifetime()
	if lifetime == SingletonLifetime {
		scope = svcScope
	} else if lifetime == ScopedLifetime && scope == svcScope {
		return nil, errors.New("scoped service must be resolved from a child scope")
	}

	// For Singleton or Scoped services, we store the result.
	// See if this service has already been resolved.
	if lifetime != TransientLifetime {
		scope.resolvedMu.RLock()
		res, exists := scope.resolved[svc.Key()]
		scope.resolvedMu.RUnlock()

		if exists {
			return res.val, res.err
		}
	}

	// Throw an error if we've already visited this service
	if ok := visitor.Enter(key); !ok {
		return nil, ErrDependencyCycle
	}
	defer visitor.Leave(key)

	// Recursively resolve dependencies
	var depVals []reflect.Value

	deps := svc.Dependencies()
	if len(deps) > 0 {
		depVals = make([]reflect.Value, len(deps))
		for i, depKey := range deps {
			var depVal any
			var depErr error

			switch depKey.Type {
			case typeContext:
				// Pass along the context
				depVal = ctx

			case typeScope:
				var ready func()
				depVal, ready = newInjectedScope(scope, key)
				defer ready()

			default:
				// Recursive call
				depVal, depErr = resolve(ctx, scope, depKey, visitor)
			}

			if depErr != nil {
				// Stop at the first error
				return nil, errors.Wrapf(depErr, "dependency %s", depKey)
			}
			depVals[i] = safeVal(depKey.Type, depVal)
		}
	}

	// Get decorator dependencies ready
	// decorators will be applied after the service is created
	var decoratorDeps [][]reflect.Value
	decorators := scope.decorators[key]
	if len(decorators) > 0 {
		decoratorDeps = make([][]reflect.Value, len(decorators))

		for i, dec := range decorators {
			decoratorDeps[i] = make([]reflect.Value, len(dec.deps))

			for j, depKey := range dec.deps {
				var depVal any
				var depErr error

				switch {
				case depKey == key:
					// We need to set this after the service is created
					continue

				case depKey.Type == typeContext:
					// Pass along the context
					depVal = ctx

				case depKey.Type == typeScope:
					var ready func()
					depVal, ready = newInjectedScope(scope, key)
					defer ready()

				default:
					// Recursive call
					depVal, depErr = resolve(ctx, scope, depKey, visitor)
				}

				if depErr != nil {
					return nil, errors.Wrapf(depErr, "decorator %s: dependency %s", dec, depKey)
				}
				decoratorDeps[i][j] = safeVal(depKey.Type, depVal)
			}
		}
	}

	if svc.Lifetime() != TransientLifetime {
		// We need to lock before we create the service to make sure we don't create it twice
		scope.resolvedMu.Lock()
		defer scope.resolvedMu.Unlock()

		// Check if another goroutine resolved the service since the last check
		if res, exists := scope.resolved[svc.Key()]; exists {
			return res.val, res.err
		}

		defer func() {
			// Store the result
			scope.resolved[svc.Key()] = resolveResult{val, err}
		}()
	}

	// Create the service
	val, err = svc.New(depVals)

	// Skip the rest if there was an error
	if err != nil {
		return val, err
	}

	// Apply decorators
	for i, dec := range decorators {
		for j, depKey := range dec.deps {
			if depKey == key {
				// Inject the service being decorated
				decoratorDeps[i][j] = safeVal(key.Type, val)
			}
		}

		val = dec.Decorate(decoratorDeps[i])
	}

	// Add Closer for the service
	if closer := svc.CloserFor(val); closer != nil {
		scope.closersMu.Lock()
		scope.closers = append(scope.closers, closer)
		scope.closersMu.Unlock()
	}

	return val, nil
}

// Close the [Container] and resolved services.
//
// Services are closed in the reverse order they were resolved/created.
// Errors returned from closing services are joined together.
//
// Close will return an error if called more than once.
func (c *Container) Close(ctx context.Context) error {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()

	if c.closed {
		return errors.Wrap(ErrContainerClosed, "di.Container.Close: closed already")
	}
	c.closed = true

	// Close services in LIFO order
	// This is important because of dependencies
	var errs []error
	for i := len(c.closers) - 1; i >= 0; i-- {
		err := c.closers[i].Close(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return errors.Wrap(err, "di.Container.Close")
	}

	return nil
}

var (
	// ErrServiceNotRegistered is returned when a service is not registered.
	ErrServiceNotRegistered = errors.New("service not registered")

	// ErrDependencyCycle is returned when a dependency cycle is detected.
	ErrDependencyCycle = errors.New("dependency cycle detected")

	// ErrContainerClosed is returned when the container is closed.
	ErrContainerClosed = errors.New("container closed")
)

type optionOrder int8

const (
	orderService   optionOrder = iota
	orderDecorator optionOrder = iota
)

func newContainerOption(order optionOrder, fn func(*Container) error) ContainerOption {
	return containerOption{fn: fn, ord: order}
}

type containerOption struct {
	fn  func(*Container) error
	ord optionOrder
}

func (o containerOption) order() optionOrder {
	return o.ord
}

func (o containerOption) applyContainer(c *Container) error {
	return o.fn(c)
}

type resolveResult struct {
	val any
	err error
}

type resolveVisitor map[serviceKey]struct{}

func (v resolveVisitor) Enter(key serviceKey) bool {
	if _, exists := v[key]; exists {
		return false
	}

	v[key] = struct{}{}
	return true
}

func (v resolveVisitor) Leave(key serviceKey) {
	delete(v, key)
}
