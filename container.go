package di

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/sectrean/di-kit/internal/errors"
)

// Container is a dependency injection container.
// It is used to resolve services by first resolving their dependencies.
type Container struct {
	parent     *Container
	services   map[serviceKey][]service
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
//   - [WithDependencyValidation] validates service dependencies.
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
		Tag:  sc.Tag(),
	}
	c.services[key] = append(c.services[key], sc)
}

// WithDependencyValidation validates registered services on [Container] creation.
//
// This will check that all dependencies are registered and that there are no dependency cycles.
// It will return an error with details if any issues are found.
//
// Scoped services are not validated because depedencies may be registered with a child scope.
func WithDependencyValidation() ContainerOption {
	return newContainerOption(orderValidation, func(c *Container) error {
		err := c.validateDependencies()
		if err != nil {
			return errors.Wrap(err, "WithDependencyValidation")
		}

		return nil
	})
}

func (c *Container) validateDependencies() error {
	var errs []error
	svcProblems := make(map[service]string)

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

func (c *Container) validateService(svc service, svcProblems map[service]string, visitor resolveVisitor) string {
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

		if depKey.Type.Kind() == reflect.Slice {
			if svc.(*funcService).IsVariadic() {
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

func (c *Container) lookupService(key serviceKey) service {
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
// Services registered with the parent [Container] will be inherited by the child [Container].
// Additional services can be registered with the new scope if needed and they will be isolated from
// the parent and sibling containers.
//
// Available options:
//   - [WithService] registers a service with a value or a function.
//   - [WithDependencyValidation] validates service dependencies.
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

	for scope := c; scope != nil; scope = scope.parent {
		if _, found := scope.services[key]; found {
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
	if key.Type.Kind() == reflect.Slice {
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
				optional := false
				if i == len(deps)-1 && svc.(*funcService).IsVariadic() {
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
	orderService    optionOrder = iota
	orderValidation optionOrder = iota
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
