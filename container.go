package di

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// NewContainer creates a new Container with the given options.
func NewContainer(opts ...ContainerOption) (*Container, error) {
	options := new(ContainerOptions)
	var errs []error
	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	// The last service registered for a type will win
	// TODO: Register a special service that returns a slice of services
	services := make(map[reflect.Type]Service)
	for _, s := range options.services {
		services[s.Type()] = s
	}

	c := &Container{
		parent:    options.parent,
		services:  services,
		resolved:  make(map[reflect.Type]resolvedService),
		resolveMu: new(sync.Mutex),
		closers:   make([]Closer, 0),
		closed:    new(atomic.Bool),
	}
	return c, nil
}

// Container allows you to resolve services and invoke functions with dependencies.
type Container struct {
	noCopy noCopy
	parent *Container

	services  map[reflect.Type]Service
	resolved  map[reflect.Type]resolvedService
	resolveMu *sync.Mutex

	closers []Closer
	closed  *atomic.Bool
}

type resolvedService struct {
	val any
	err error
}

// HasType returns true if the container has a service of the given type.
func (c *Container) HasType(typ reflect.Type) bool {
	_, exists := c.services[typ]

	if !exists && c.parent != nil {
		return c.parent.HasType(typ)
	}
	return exists
}

func wrapResolveError(typ reflect.Type, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("resolving type %s: %w", typ, err)
}

// Resolve returns a service of the given type.
//
// The type must be registered with the container.
func (c *Container) Resolve(ctx context.Context, typ reflect.Type) (any, error) {
	// Resolve cannot be called after a container has been closed
	if c.closed.Load() {
		panic("cannot resolve type after container has been closed")
	}

	// Check if the context has been closed
	if ctx.Err() != nil {
		return nil, wrapResolveError(typ, ctx.Err())
	}

	// Check if the type is a special type
	if val, ok := c.getSpecialType(ctx, typ); ok {
		return val, nil
	}

	// Check if we've already resolved this service
	if rs, ok := c.resolved[typ]; ok {
		return rs.val, wrapResolveError(typ, rs.err)
	}

	// TODO: Benchmark concurrent Resolve calls
	// Then see if we can optimize it.
	// Maybe we can use sync.Map and sync.OnceValue() to guarantee that each service is created
	// only once.
	// We need to think about a possible deadlock if a service injects a Scope and
	// then calls Resolve() in the constructor function.
	c.resolveMu.Lock()
	defer c.resolveMu.Unlock()

	// One last check now that we've acquired the mutex to see if the service has been resolved already
	if rs, ok := c.resolved[typ]; ok {
		return rs.val, wrapResolveError(typ, rs.err)
	}

	// Recursively resolve the type and its dependencies
	visitor := newResolveVisitor()
	val, err := c.resolve(ctx, typ, visitor)

	return val, wrapResolveError(typ, err)
}

func (c *Container) getSpecialType(ctx context.Context, typ reflect.Type) (any, bool) {
	switch typ {
	case typContext:
		return ctx, true
	case typScope:
		return c, true
	}
	return nil, false
}

func (c *Container) resolve(ctx context.Context, typ reflect.Type, visitor *resolveVisitor) (any, error) {
	// Check if the type is a special type
	if val, ok := c.getSpecialType(ctx, typ); ok {
		return val, nil
	}

	// Check if we've already resolved this service
	if rs, ok := c.resolved[typ]; ok {
		return rs.val, rs.err
	}

	// Check if the type is registered
	svc, ok := c.services[typ]
	if !ok {
		if c.parent != nil && c.parent.HasType(typ) {
			return c.parent.Resolve(ctx, typ)
		}
		return nil, ErrTypeNotRegistered
	}

	// Throw an error if we've already visited this service
	if visited := visitor.Enter(typ); visited {
		return nil, ErrDependencyCycle
	}
	defer visitor.Leave(typ)

	// Recursively resolve dependencies
	// Stop at the first error
	deps := make([]any, len(svc.Dependencies()))
	for i, depTyp := range svc.Dependencies() {
		depVal, depErr := c.resolve(ctx, depTyp, visitor)
		if depErr != nil {
			return depVal, fmt.Errorf("resolving dependency %s: %w", depTyp, depErr)
		}
		deps[i] = depVal
	}

	// Check context for errors before creating the service
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create the instance and cache the service and error as-is
	val, err := svc.GetValue(deps)
	c.resolved[typ] = resolvedService{val, err}

	return val, err
}

// Invoke calls the given function with dependencies resolved from the container.
//
// The function may take any number of arguments. These dependencies must be registered with the container.
// The function may also accept a context.Context.
// The function may return an error.
func (c *Container) Invoke(ctx context.Context, fn any) error {
	fnTyp := reflect.TypeOf(fn)
	fnVal := reflect.ValueOf(fn)

	// Make sure fn is a function
	if fnTyp.Kind() != reflect.Func {
		panic("fn must be a function")
	}

	// Invoke should never be called after a container has been closed
	if c.closed.Load() {
		panic("cannot invoke fn after container has been closed")
	}

	// Resolve fn arguments from the container
	// Stop at the first error
	numIn := fnTyp.NumIn()
	in := make([]reflect.Value, numIn)

	for i := 0; i < numIn; i++ {
		argTyp := fnTyp.In(i)
		argVal, argErr := c.Resolve(ctx, argTyp)
		if argErr != nil {
			return fmt.Errorf("resolving fn argument: %w", argErr)
		}
		in[i] = reflect.ValueOf(argVal)
	}

	// Check for a context error before the function is invoked
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Invoke the function
	out := fnVal.Call(in)

	// Check if the function returns an error; ignore any other return values
	var err error
	for i := 0; i < fnTyp.NumOut(); i++ {
		if fnTyp.Out(i) == typError {
			if !out[i].IsNil() {
				err = out[i].Interface().(error)
			}
			break
		}
	}

	// Don't wrap this error; return it as-is
	return err
}

// Closed returns true if the container has been closed.
func (c *Container) Closed() bool {
	return c.closed.Load()
}

// Close closes the container and all of its services.
func (c *Container) Close(ctx context.Context) error {
	// Close can only be called once
	if c.closed.Swap(true) {
		panic("container already closed")
	}

	// Take the resolve lock so no more services can be resolved
	c.resolveMu.Lock()
	defer c.resolveMu.Unlock()

	// TODO: Should we track child scopes to make sure all child scopes have been closed?

	// Services are created in dependency order, so we need to close them in the reverse order
	// Join all returned errors
	var errs []error
	for i := len(c.closers) - 1; i >= 0; i-- {
		err := c.closers[i].Close(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}
	err := errors.Join(errs...)
	if err != nil {
		return fmt.Errorf("closing container services: %w", err)
	}

	return nil
}

var _ Scope = &Container{}
var _ Closer = &Container{}

type resolveVisitor struct {
	// This could be a set
	visited map[reflect.Type]bool
}

func newResolveVisitor() *resolveVisitor {
	return &resolveVisitor{
		visited: make(map[reflect.Type]bool),
	}
}

// Enter returns true if the service has already been visited
func (v *resolveVisitor) Enter(typ reflect.Type) bool {
	if _, exists := v.visited[typ]; exists {
		return true
	}

	v.visited[typ] = true
	return false
}

func (v *resolveVisitor) Leave(typ reflect.Type) {
	delete(v.visited, typ)
}

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
