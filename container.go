package di

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/sectrean/di-kit/internal/errors"
)

// Container is a dependency injection container.
// It is used to resolve services by first resolving their dependencies.
type Container struct {
	parent     *Container
	services   map[serviceKey][]*service
	resolved   map[*service]resolveResult
	closers    []Closer
	resolvedMu sync.RWMutex
	closedMu   sync.RWMutex
	closersMu  sync.Mutex
	closed     bool
	validate   bool
}

var _ Scope = (*Container)(nil)

// NewContainer creates a new [Container] with the provided options.
//
// Available options:
//   - [WithService] registers a service with a value or constructor function.
//   - [WithModule] registers services from a module.
//   - [WithDependencyValidation] validates service dependencies.
func NewContainer(opts ...ContainerOption) (*Container, error) {
	c := &Container{
		services: make(map[serviceKey][]*service),
		resolved: make(map[*service]resolveResult),
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
	applyContainer(*Container) error
}

type containerOption func(*Container) error

func (o containerOption) applyContainer(c *Container) error {
	return o(c)
}

func (c *Container) applyOptions(opts []ContainerOption) error {
	err := applyOptions(opts, func(o ContainerOption) error {
		return o.applyContainer(c)
	})
	if err != nil {
		return err
	}

	if c.validate {
		err := c.validateDependencies()
		if err != nil {
			return errors.Wrap(err, "WithDependencyValidation")
		}
	}

	return nil
}

func (c *Container) register(s *service) {
	if c.services == nil {
		c.services = make(map[serviceKey][]*service)
	}

	if len(s.Assignables()) == 0 {
		c.registerType(s.Type(), s)
	} else {
		for _, assignable := range s.Assignables() {
			c.registerType(assignable, s)
		}
	}

	// Add closers for value services
	// We don't need to take locks here because this is only called when creating a new Container
	if s.IsValue() {
		if closer := s.CloserFor(s.Value()); closer != nil {
			c.closers = append(c.closers, closer)
		}
	}
}

func (c *Container) registerType(t reflect.Type, s *service) {
	if len(s.Tags()) == 0 {
		key := serviceKey{
			Type: t,
		}
		c.services[key] = append(c.services[key], s)
	} else {
		// This doesn't de-duplicate tags, so if someone registers duplicate tags, that's on them
		for _, tag := range s.Tags() {
			key := serviceKey{
				Type: t,
				Tag:  tag,
			}
			c.services[key] = append(c.services[key], s)
		}
	}
}

// WithDependencyValidation validates registered services on [Container] creation.
//
// This will check that all dependencies are registered and that there are no dependency cycles.
// It will return an error with details if any issues are found.
//
// Scoped services are not validated because dependencies may be registered with a child scope.
// They can be validated using this option when creating a child scope with [Container.NewScope].
func WithDependencyValidation() ContainerOption {
	return containerOption(func(c *Container) error {
		c.validate = true
		return nil
	})
}

func (c *Container) validateDependencies() error {
	var errs []error
	svcProblems := make(map[*service]string)

	for _, svcs := range c.services {
		for _, svc := range svcs {
			if svc.Lifetime() == ScopedLifetime {
				// Scoped services are not validated
				continue
			}

			prob := c.validateService(svc, svcProblems, make(resolveVisitor))
			if prob != "" {
				errs = append(errs, errors.Errorf("service %s: %s", svc, prob))
			}
		}
	}

	if c.parent != nil {
		// Validate scoped services on the parent Container
		for _, svcs := range c.parent.services {
			for _, svc := range svcs {
				if svc.Lifetime() != ScopedLifetime {
					// Now we only want the scoped services
					continue
				}

				prob := c.validateService(svc, svcProblems, make(resolveVisitor))
				if prob != "" {
					errs = append(errs, errors.Errorf("service %s: %s", svc, prob))
				}
			}
		}
	}

	return errors.Join(errs...)
}

func (c *Container) validateService(svc *service, svcProblems map[*service]string, visitor resolveVisitor) string {
	if prob, ok := svcProblems[svc]; ok {
		return prob
	}

	deps := svc.Dependencies()
	if len(deps) == 0 {
		svcProblems[svc] = ""
		return ""
	}

	if !visitor.Enter(svc) {
		return errDependencyCycle.Error()
	}
	defer visitor.Leave(svc)

	var problems []string
	for _, depKey := range deps {
		if depKey.Type == typeContext || depKey.Type == typeScope {
			continue
		}

		if isUnnamedSliceType(depKey.Type) {
			if svc.Func().Type().IsVariadic() {
				// If the service is variadic, registration is optional
				continue
			}

			// Check that the element type is registered
			depKey.Type = depKey.Type.Elem()
		}

		depSvc := c.lookupService(depKey)
		if depSvc == nil {
			prob := fmt.Sprintf("dependency %s: service not registered", depKey)
			problems = append(problems, prob)
			continue
		}

		prob := c.validateService(depSvc, svcProblems, visitor)
		if prob != "" {
			problems = append(problems, fmt.Sprintf("dependency %s: %s", depKey, prob))
		}
	}

	if len(problems) > 0 {
		probs := strings.Join(problems, "; ")
		svcProblems[svc] = probs
		return probs
	}

	return ""
}

func (c *Container) lookupService(key serviceKey) *service {
	for scope := c; scope != nil; scope = scope.parent {
		svcs, ok := scope.services[key]
		if !ok {
			continue
		}

		// Return the last registered service for this key
		return svcs[len(svcs)-1]
	}

	return nil
}

// NewScope creates a new [Container] with a child scope.
//
// Services registered with the parent container will be inherited by the child.
// For services registered with [ScopedLifetime], each child container will create an isolated instance
// when the service is resolved.
//
// Additional services can be registered when creating the new scope if needed and they will be isolated from
// the parent and sibling containers.
//
// Available options:
//   - [WithService] registers a service with a value or a function.
//   - [WithModule] registers services from a module.
//   - [WithDependencyValidation] validates service dependencies.
func (c *Container) NewScope(opts ...ContainerOption) (*Container, error) {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()

	if c.closed {
		return nil, errors.Wrap(errContainerClosed, "di.Container.NewScope")
	}

	scope := &Container{
		parent:   c,
		resolved: make(map[*service]resolveResult),
	}

	err := scope.applyOptions(opts)
	if err != nil {
		return nil, errors.Wrap(err, "di.Container.NewScope")
	}

	return scope, nil
}

// Contains returns true if the container has a service registered for the given [reflect.Type].
//
// Available options:
//   - [WithTag] specifies a key associated with the service.
func (c *Container) Contains(t reflect.Type, opts ...ResolveOption) bool {
	// Check if the type is a slice, look for the element type
	if isUnnamedSliceType(t) {
		t = t.Elem()
	}

	key := serviceKey{Type: t}
	for _, opt := range opts {
		key = opt.applyServiceKey(key)
	}

	for scope := c; scope != nil; scope = scope.parent {
		if _, found := scope.services[key]; found {
			return true
		}
	}

	return false
}

// ResolveOption can be used when calling [Resolve], [MustResolve],
// [Container.Resolve], or [Container.Contains].
type ResolveOption interface {
	// applyServiceKey does not use a pointer to optimize allocations.
	applyServiceKey(serviceKey) serviceKey
}

// Resolve a service of the given [reflect.Type].
//
// This will return an error under the following conditions:
//   - The container has been closed
//   - The type is not registered with the container
//   - The type cannot be resolved due to unregistered dependencies
//   - A dependency cycle is detected
//   - A service's constructor function returns an error
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

	val, err := resolveKey(ctx, c, key, make(resolveVisitor), false)
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
	optional bool,
) (any, error) {
	if isUnnamedSliceType(key.Type) {
		return resolveSliceKey(ctx, scope, key, visitor, optional)
	}

	// Look up the service
	svc := scope.lookupService(key)
	if svc == nil {
		// If the service is not found, return an error
		// TODO: Support optional dependencies?
		return nil, errServiceNotRegistered
	}

	return resolveService(ctx, scope, key, svc, visitor)
}

func resolveSliceKey(
	ctx context.Context,
	scope *Container,
	key serviceKey,
	visitor resolveVisitor,
	optional bool,
) (any, error) {
	sliceVal := reflect.MakeSlice(key.Type, 0, 0)
	elemType := key.Type.Elem()
	elemKey := serviceKey{
		Type: elemType,
		Tag:  key.Tag,
	}
	found := false

	for s := scope; s != nil; s = s.parent {
		for _, svc := range s.services[elemKey] {
			val, err := resolveService(ctx, scope, elemKey, svc, visitor)
			if err != nil {
				return nil, err
			}

			sliceVal = reflect.Append(sliceVal, safeReflectValue(elemType, val))
			found = true
		}
	}

	if !found && !optional {
		// If the service is not found, return an error
		return nil, errServiceNotRegistered
	}

	return sliceVal.Interface(), nil
}

func resolveService(
	ctx context.Context,
	scope *Container,
	key serviceKey,
	svc *service,
	visitor resolveVisitor,
) (val any, err error) {
	if svc.IsValue() {
		// Value services are always resolved, so we can return the value directly.
		return svc.Value(), nil
	}

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
			return res.Val, res.Err
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
				optional := false
				if i == len(deps)-1 && svc.Func().Type().IsVariadic() {
					// If this is the last arg and the constructor function is variadic,
					// we treat it as optional.
					optional = true
				}

				// Recursive call
				depVal, depErr = resolveKey(ctx, scope, depKey, visitor, optional)
			}

			if depErr != nil {
				// Stop at the first error
				return nil, errors.Wrapf(depErr, "dependency %s", depKey)
			}
			depVals[i] = safeReflectValue(depKey.Type, depVal)
		}
	}

	if svc.Lifetime() != TransientLifetime {
		// We need to lock before we create the service to make sure we don't create it twice
		scope.resolvedMu.Lock()
		defer scope.resolvedMu.Unlock()

		// Check if another goroutine resolved the service since the last check
		if res, exists := scope.resolved[svc]; exists {
			return res.Val, res.Err
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

	// Add Closer for the service
	if closer := svc.CloserFor(val); closer != nil {
		scope.closersMu.Lock()
		scope.closers = append(scope.closers, closer)
		scope.closersMu.Unlock()
	}

	return val, nil
}

// Close all services resolved by this container.
// See [Closer] for more information.
//
// Services are closed in the reverse order they were resolved/created.
// Errors returned from closing services are joined together.
//
// Resolve and NewScope will return an error if called after the container has been closed.
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

type resolveResult struct {
	Val any
	Err error
}

type resolveVisitor map[*service]struct{}

func (v resolveVisitor) Enter(s *service) bool {
	if _, exists := v[s]; exists {
		return false
	}

	v[s] = struct{}{}
	return true
}

func (v resolveVisitor) Leave(s *service) {
	delete(v, s)
}
