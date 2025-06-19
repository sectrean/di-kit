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
	services   map[serviceKey][]service
	decorators map[serviceKey][]*decorator
	resolved   map[service]resolveResult
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
		services: make(map[serviceKey][]service),
		resolved: make(map[service]resolveResult),
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
	opts = flattenModules(opts)

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
	if c.services == nil {
		c.services = make(map[serviceKey][]service)
	}

	if len(sc.Assignables()) == 0 {
		c.registerType(sc.Type(), sc)
	} else {
		for _, assignable := range sc.Assignables() {
			c.registerType(assignable, sc)
		}
	}

	// Add closers for value services
	// We don't need to take locks here because this is only called when creating a new Container
	if vs, ok := sc.(*valueService); ok {
		if closer := sc.CloserFor(vs.val); closer != nil {
			c.closers = append(c.closers, closer)
		}
	}
}

func (c *Container) registerType(t reflect.Type, sc serviceConfig) {
	key := serviceKey{
		Type: t,
	}
	c.services[key] = append(c.services[key], sc)

	// Register the service with a tag if it has one
	if sc.Tag() != nil {
		keyWithTag := serviceKey{
			Type: t,
			Tag:  sc.Tag(),
		}
		c.services[keyWithTag] = append(c.services[keyWithTag], sc)
	}
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
		return nil, errors.Wrap(errContainerClosed, "di.Container.NewScope")
	}

	scope := &Container{
		parent:   c,
		resolved: make(map[service]resolveResult),
	}

	err := scope.applyOptions(opts)
	if err != nil {
		return nil, errors.Wrap(err, "di.Container.NewScope")
	}

	return scope, nil
}

// Contains returns true if the [Container] has a service registered for the given [reflect.Type].
//
// Available options:
//   - [WithTag] specifies a key associated with the service.
func (c *Container) Contains(t reflect.Type, opts ...ResolveOption) bool {
	// Check if the type is a slice, look for the element type
	if t.Kind() == reflect.Slice {
		t = t.Elem()
	}

	key := serviceKey{Type: t}
	for _, opt := range opts {
		key = opt.applyServiceKey(key)
	}

	for s := c; s != nil; s = s.parent {
		if _, found := s.services[key]; found {
			return true
		}
	}

	return false
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
		return nil, errors.Wrapf(errContainerClosed, "di.Container.Resolve %s", key)
	}

	val, err := resolveKey(ctx, c, key, make(resolveVisitor))
	if err != nil {
		return val, errors.Wrapf(err, "di.Container.Resolve %s", key)
	}

	return val, nil
}

func resolveKey(
	ctx context.Context,
	scope *Container,
	key serviceKey,
	visitor resolveVisitor,
) (any, error) {
	if key.Type.Kind() == reflect.Slice {
		return resolveSliceKey(ctx, scope, key, visitor)
	}

	// Look up the service
	var svc service
	for s := scope; s != nil; s = s.parent {
		svcs, ok := s.services[key]
		if ok {
			// The last service registered for a type will win
			svc = svcs[len(svcs)-1]
			break
		}
	}

	if svc == nil {
		return nil, errServiceNotRegistered
	}

	return resolveService(ctx, scope, key, svc, visitor)
}

func resolveSliceKey(
	ctx context.Context,
	scope *Container,
	key serviceKey,
	visitor resolveVisitor,
) (any, error) {
	sliceVal := reflect.MakeSlice(key.Type, 0, 0)
	elementKey := serviceKey{
		Type: key.Type.Elem(),
		Tag:  key.Tag,
	}
	found := false

	for s := scope; s != nil; s = s.parent {
		for _, svc := range s.services[elementKey] {
			val, err := resolveService(ctx, scope, elementKey, svc, visitor)
			if err != nil {
				return nil, err
			}
			if val != nil {
				sliceVal = reflect.Append(sliceVal, reflect.ValueOf(val))
			}

			found = true
		}
	}

	if !found {
		return nil, errServiceNotRegistered
	}

	return sliceVal.Interface(), nil
}

func resolveService(
	ctx context.Context,
	scope *Container,
	key serviceKey,
	svc service,
	visitor resolveVisitor,
) (val any, err error) {
	// Check context for errors
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// For singleton services, use the scope the service is registered with.
	// Otherwise, use the current scope.
	lifetime := svc.Lifetime()
	if lifetime == SingletonLifetime {
		scope = svc.Scope()
	} else if lifetime == ScopedLifetime && scope == svc.Scope() {
		return nil, errors.New("scoped service must be resolved from a child scope")
	}

	// For Singleton or Scoped services, we store the result.
	// See if this service has already been resolved.
	if lifetime != TransientLifetime {
		scope.resolvedMu.RLock()
		res, exists := scope.resolved[svc]
		scope.resolvedMu.RUnlock()

		if exists {
			return res.val, res.err
		}
	}

	// Throw an error if we've already visited this service
	if !visitor.Enter(svc) {
		return nil, errDependencyCycle
	}
	defer visitor.Leave(svc)

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
				depVal, depErr = resolveKey(ctx, scope, depKey, visitor)
			}

			if depErr != nil {
				// Stop at the first error
				return nil, errors.Wrapf(depErr, "dependency %s", depKey)
			}
			depVals[i] = safeReflectValue(depKey.Type, depVal)
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
					depVal, depErr = resolveKey(ctx, scope, depKey, visitor)
				}

				if depErr != nil {
					return nil, errors.Wrapf(depErr, "decorator %s: dependency %s", dec, depKey)
				}
				decoratorDeps[i][j] = safeReflectValue(depKey.Type, depVal)
			}
		}
	}

	if svc.Lifetime() != TransientLifetime {
		// We need to lock before we create the service to make sure we don't create it twice
		scope.resolvedMu.Lock()
		defer scope.resolvedMu.Unlock()

		// Check if another goroutine resolved the service since the last check
		if res, exists := scope.resolved[svc]; exists {
			return res.val, res.err
		}

		defer func() {
			// Store the result
			scope.resolved[svc] = resolveResult{val, err}
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
				decoratorDeps[i][j] = safeReflectValue(key.Type, val)
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
		return errors.Wrap(errContainerClosed, "di.Container.Close: closed already")
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
	errServiceNotRegistered = errors.New("service not registered")
	errDependencyCycle      = errors.New("dependency cycle detected")
	errContainerClosed      = errors.New("container closed")
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

type resolveVisitor map[service]struct{}

func (v resolveVisitor) Enter(s service) bool {
	if _, exists := v[s]; exists {
		return false
	}

	v[s] = struct{}{}
	return true
}

func (v resolveVisitor) Leave(s service) {
	delete(v, s)
}
