package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type Container interface {
	NewChild() Container
	Resolve(context.Context, reflect.Type) (any, error)
	Invoke(context.Context, any) error
	Close(context.Context) error
}

func Resolve[T any](ctx context.Context, c Container) (T, error) {
	t := TypeOf[T]()
	val, err := c.Resolve(ctx, t)
	if err != nil {
		// We can't convert a nil value to T
		// so we return the zero value of T
		var zeroVal T
		return zeroVal, err
	}

	return val.(T), err
}

func MustResolve[T any](ctx context.Context, c Container) T {
	val, err := Resolve[T](ctx, c)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	return val
}

type container struct {
	root        *container
	services    map[reflect.Type]Service
	instances   map[reflect.Type]any
	instancesMu *sync.Mutex
	closed      *atomic.Bool
	closers     []Closer
	closersMu   *sync.Mutex
}

// NewContainer creates a new root container
func NewContainer(opts ...ContainerOption) (Container, error) {
	options := &ContainerOptions{}

	var result error
	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}
	if result != nil {
		return nil, errors.Wrap(result, "creating root container")
	}

	// The last service registered for a type will win
	// TODO: Register a special service that returns a slice of all services
	services := make(map[reflect.Type]Service)
	for _, s := range options.Services {
		services[s.Type()] = s
	}

	// New root Container
	root := newScope(nil, services)
	return root, nil
}

func newScope(root *container, services map[reflect.Type]Service) *container {
	return &container{
		root:        root,
		services:    services,
		instances:   make(map[reflect.Type]any),
		instancesMu: &sync.Mutex{},
		closed:      &atomic.Bool{},
		closers:     make([]Closer, 0),
		closersMu:   &sync.Mutex{},
	}
}

// isRoot returns true if this is the root container
// TODO: Should we make this function public?
func (c *container) isRoot() bool {
	return c.root == nil
}

func (c *container) getRoot() *container {
	if c.root != nil {
		return c.root
	}
	return c
}

// NewChild creates a new container with a child scope
func (c *container) NewChild() Container {
	child := newScope(c.getRoot(), c.services)
	return child
}

func (c *container) Resolve(ctx context.Context, t reflect.Type) (any, error) {
	const errMsgFormat = "cannot resolve type %v"

	// Resolve should never be called after a container has been closed
	if c.closed.Load() {
		// TODO: Should this be a panic?
		return nil, errors.Wrapf(ErrContainerClosed, errMsgFormat, t)
	}

	if t == tContext {
		return ctx, nil
	}
	if t == tContainer {
		return c, nil
	}

	svc, ok := c.services[t]
	if !ok {
		return nil, errors.Wrapf(ErrTypeNotRegistered, errMsgFormat, t)
	}

	visitor := newResolveVisitor(c)

	// Lock the root container
	// This is the easiest way to make this concurrency safe,
	// but performance may be a concern with many concurrent goroutines
	// TODO: Benchmark this function and optimize locking for better performance
	c.getRoot().instancesMu.Lock()
	defer c.getRoot().instancesMu.Unlock()

	val, err := resolveInstance(ctx, svc, visitor)

	// Wrap error to include type information and stack trace
	err = errors.Wrapf(err, errMsgFormat, t)

	return val, err
}

// resolveInstance recursively resolves the dependencies for a service
// and returns the value.
func resolveInstance(ctx context.Context, svc Service, visitor *resolveVisitor) (any, error) {
	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Work with the root container if the service is singleton
	var scope = visitor.Scope

	if svc.Lifetime() == Singleton {
		scope = visitor.Root
	}

	// Check if we've already resolved this service
	if val, ok := scope.instances[svc.Type()]; ok {
		return val, nil
	}

	// Detect circular dependencies
	// Throw an error if we've already visited this service
	if visited := visitor.Enter(svc); visited {
		// We add the dependency chain to the error message for debugging
		trail := visitor.Trail(svc.Type())
		return nil, errors.WithMessagef(ErrCircularDependency,
			"circular dependency detected resolving type %v: %s", svc.Type(), trail)
	}

	var deps []any

	// Recursively resolve dependencies
	for _, t := range svc.Dependencies() {
		// If one of the dependencies is context.Context, use the context
		if t == tContext {
			deps = append(deps, ctx)
			continue
		}
		// TODO: Which scope should be used for dependencies?
		if t == tContainer {
			return scope, nil
		}

		dep, ok := scope.services[t]
		if !ok {
			return nil, errors.WithMessagef(ErrTypeNotRegistered, "type %v not registered", t)
		}

		val, err := resolveInstance(ctx, dep, visitor)
		if err != nil {
			return nil, err
		}
		deps = append(deps, val)
	}

	// Check context for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create the instance
	// Do we want to cache errors?
	val, err := svc.GetValue(deps)

	if svc.Lifetime() != Transient {
		scope.instances[svc.Type()] = val
	}

	// TODO: I don't think we want a closer if a value was registered
	// The container should only be responsible for closing services that it created
	if closer, ok := svc.GetCloser(val); ok {
		scope.addCloser(closer)
	}

	visitor.Leave()

	return val, err
}

func (c *container) Invoke(ctx context.Context, fn any) error {
	// Make sure fn is a function
	fnVal := reflect.ValueOf(fn)
	if fnVal.Kind() != reflect.Func {
		panic("fn must be a function")
	}

	// Invoke should never be called after a container has been closed
	if alreadyClosed := c.closed.Load(); alreadyClosed {
		// TODO: Should this be a panic?
		return errors.Wrap(ErrContainerClosed, "container closed already")
	}

	// Check context for cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	var in []reflect.Value
	var resolveErr error

	// Resolve fn arguments from the container
	for i := 0; i < fnVal.Type().NumIn(); i++ {
		t := fnVal.Type().In(i)
		val, err := c.Resolve(ctx, t)
		if err != nil {
			// If the context was cancelled, return the error as-is
			if err == ctx.Err() {
				return err
			}

			multierror.Append(resolveErr, err)
		}
		in = append(in, reflect.ValueOf(val))
	}
	if resolveErr != nil {
		return errors.Wrap(resolveErr, "resolving fn argument(s)")
	}

	// Check context for cancellation one last time
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Invoke the function
	var err error
	out := fnVal.Call(in)

	// Check if the function returns an error
	// Ignore any other return values
	// If for some reason multiple errors are returned, we only return the first one
	for i := 0; i < fnVal.Type().NumOut(); i++ {
		t := fnVal.Type().Out(i)
		if t == tError {
			if !out[i].IsNil() {
				// Don't wrap the error; return it as-is
				err = out[i].Interface().(error)
			}
			break
		}
	}

	return err
}

func (c *container) addCloser(closer Closer) {
	c.closersMu.Lock()
	c.closers = append(c.closers, closer)
	c.closersMu.Unlock()
}

func (c *container) Close(ctx context.Context) error {
	// Close should only be called once
	if c.closed.Swap(true) {
		var panicMsg string
		if c.isRoot() {
			panicMsg = "Close was called more than once on the root Container"
		} else {
			panicMsg = "Close was called more than once on a child Container"
		}
		panic(panicMsg)
	}

	// TODO: Should we track child scopes to make sure all child scopes have been closed?

	// Call all Closers in the opposite order they were added
	// Aggregate any returned errors
	var result error
	for i := len(c.closers) - 1; i >= 0; i-- {
		err := c.closers[i].Close(ctx)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}

	return errors.Wrap(result, "closing container")
}
