package di

import (
	"context"
	"reflect"
	"sync"

	"github.com/johnrutherford/di-kit/internal/errors"
	"github.com/puzpuzpuz/xsync/v3"
)

// NewContainer creates a new Container with the provided options.
//
// Available options:
//   - [Register] registers a service with a value or a function.
func NewContainer(opts ...ContainerOption) (*Container, error) {
	c := &Container{
		resolved: xsync.NewMapOf[service, *resolveFuture](),
		closeMu:  xsync.NewRBMutex(),
	}

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

// Container is a dependency injection container.
// It is used to resolve services by first resolving their dependencies.
type Container struct {
	parent   *Container
	services map[serviceKey]service

	resolved *xsync.MapOf[service, *resolveFuture]

	closersMu sync.Mutex
	closers   []Closer

	closeMu *xsync.RBMutex
	closed  bool
}

var (
	// ErrTypeNotRegistered is returned when a type is not registered.
	ErrTypeNotRegistered = errors.New("type not registered")

	// ErrDependencyCycle is returned when a dependency cycle is detected.
	ErrDependencyCycle = errors.New("dependency cycle detected")

	// ErrContainerClosed is returned when the container is closed.
	ErrContainerClosed = errors.New("container closed")
)

var _ Scope = (*Container)(nil)

// Register registers the provided service.
func (c *Container) register(s service) {
	c.initServices()

	if len(s.Aliases()) == 0 {
		c.registerType(s.Type(), s)
	} else {
		for _, alias := range s.Aliases() {
			c.registerType(alias, s)
		}
	}
}

func (c *Container) initServices() {
	if c.services == nil {
		c.services = make(map[serviceKey]service)
	} else if c.parent != nil && len(c.parent.services) == len(c.services) {
		// Copy the parent's services to avoid modifying it
		c.services = make(map[serviceKey]service, len(c.parent.services))
		for k, v := range c.parent.services {
			c.services[k] = v
		}
	}
}

func (c *Container) registerType(t reflect.Type, s service) {
	key := serviceKey{Type: t}

	// Use a slice service if the type is already registered
	if existing, ok := c.services[key]; ok {
		sliceKey := serviceKey{
			Type: reflect.SliceOf(t),
		}

		var sliceSvc *sliceService
		if sliceSvc, ok = c.services[sliceKey].(*sliceService); !ok {
			// Create a new slice service and register it
			sliceSvc = newSliceService(t)
			c.services[sliceKey] = sliceSvc

			// Add the existing service to the slice service
			// and register a key with a unique key
			c.services[sliceSvc.AddNewItem()] = existing
		}

		// Add the new item to slice service and register it
		c.services[sliceSvc.AddNewItem()] = s
	}

	// Register the service with a key
	if s.Key() != nil {
		keyWithKey := serviceKey{
			Type: s.Type(),
			Key:  s.Key(),
		}
		c.services[keyWithKey] = s
	}

	// The last service registered for a type will win
	c.services[key] = s
}

// NewScope creates a new Container with a child scope.
//
// Services registered with the parent Container will be inherited by the child Container.
// Additional services can be registered with the new Scope if needed.
// They will only be available to the new scope.
//
// Available options:
//   - [Register] registers a service with a value or a function.
func (c *Container) NewScope(opts ...ContainerOption) (*Container, error) {
	lock := c.closeMu.RLock()
	defer c.closeMu.RUnlock(lock)

	if c.closed {
		return nil, errors.Wrap(ErrContainerClosed, "new scope")
	}

	scope := &Container{
		parent:   c,
		services: c.services,
		resolved: xsync.NewMapOf[service, *resolveFuture](),
		closeMu:  xsync.NewRBMutex(),
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
//   - [WithKey] specifies a key associated with the service.
func (c *Container) Contains(t reflect.Type, opts ...ServiceOption) bool {
	key := serviceKey{Type: t}
	for _, opt := range opts {
		key = opt.applyServiceKey(key)
	}

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
//   - [WithKey] specifies a key associated with the service.
func (c *Container) Resolve(ctx context.Context, t reflect.Type, opts ...ServiceOption) (any, error) {
	lock := c.closeMu.RLock()
	defer c.closeMu.RUnlock(lock)

	if c.closed {
		return nil, errors.Wrapf(ErrContainerClosed, "resolve %s", t)
	}

	key := serviceKey{Type: t}
	for _, opt := range opts {
		key = opt.applyServiceKey(key)
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
	svc, ok := c.services[key]
	if !ok {
		return nil, ErrTypeNotRegistered
	}

	// For scoped services, use the current container.
	// For singleton services, use the root container.
	scope := c
	if svc.Lifetime() == Singleton {
		scope = c.root()
	}

	// Throw an error if we've already visited this service
	if visited := visitor.Enter(key); visited {
		return nil, ErrDependencyCycle
	}
	defer visitor.Leave(key)

	// For Singleton or Scoped services, we store the result
	// in a future to prevent multiple calls to the service.
	if svc.Lifetime() != Transient {
		fut, loaded := scope.resolved.LoadOrCompute(svc, newFuture)
		if loaded {
			// This will block until the value and error are set
			// by the first goroutine to resolve this service.
			return fut.Result()
		}

		defer func() {
			// Set the result when this function returns
			fut.setResult(val, err)
		}()
	}

	// Recursively resolve dependencies
	var depValues []reflect.Value
	if len(svc.Dependencies()) > 0 {
		depValues = make([]reflect.Value, len(svc.Dependencies()))
		for i, depKey := range svc.Dependencies() {
			var depVal any
			var depErr error

			switch depKey.Type {
			case contextType:
				// Pass along the context
				depVal = ctx

			case scopeType:
				// We wrap the scope to prevent Resolve from being called on it
				// until we finish resolving this service
				var ready func()
				depVal, ready = newInjectedScope(key, scope)
				defer ready()

			default:
				// Recursive call
				depVal, depErr = scope.resolve(ctx, depKey, visitor)
			}

			if depErr != nil {
				// Stop at the first error
				return depVal, errors.Wrapf(depErr, "resolve dependency %s", depKey)
			}
			depValues[i] = reflect.ValueOf(depVal)
		}
	}

	// Check context for errors before creating the service
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create the instance
	val, err = svc.GetValue(depValues)

	// Add Closer for the service
	if closer := svc.GetCloser(val); closer != nil {
		scope.appendCloser(closer)
	}

	return val, err
}

func (c *Container) appendCloser(closer Closer) {
	c.closersMu.Lock()
	c.closers = append(c.closers, closer)
	c.closersMu.Unlock()
}

// Close closes the Container and all of its services.
//
// Services are closed in the reverse order they were resolved/created.
// Errors returned from closing services are joined together.
//
// Close will return an error if called more than once.
func (c *Container) Close(ctx context.Context) error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return errors.Wrap(ErrContainerClosed, "already closed")
	}
	c.closed = true

	// Close services in reverse order
	var errs errors.MultiError
	for i := len(c.closers) - 1; i >= 0; i-- {
		err := c.closers[i].Close(ctx)
		errs = errs.Append(err)
	}

	return errs.Wrap("close container")
}

// Common types
var (
	errorType   = reflect.TypeFor[error]()
	contextType = reflect.TypeFor[context.Context]()
	scopeType   = reflect.TypeFor[Scope]()
)

type resolveFuture struct {
	val  any
	err  error
	done chan struct{}
}

func newFuture() *resolveFuture {
	return &resolveFuture{
		done: make(chan struct{}),
	}
}

func (f *resolveFuture) setResult(val any, err error) {
	f.val = val
	f.err = err
	close(f.done)
}

func (f *resolveFuture) Result() (any, error) {
	<-f.done
	return f.val, f.err
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
