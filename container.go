package di

import (
	"context"
	"iter"
	"reflect"
	"slices"
	"sync"

	"github.com/sectrean/di-kit/internal/errors"
)

// Container is a reflection-based dependency injection container.
// It is used to resolve services by first resolving their dependencies.
type Container struct {
	parent   *Container
	services map[serviceKey][]*service
	resolved map[*service]*resolveResult
	closers  []Closer

	resolvedMu sync.Mutex

	// waitingMu guards the blockedOn edges of all resolveVisitors in this container tree.
	// It is only used on the root Container.
	waitingMu sync.Mutex

	// closeMu guards closed and closers.
	// It is not held for the duration of a resolve: Resolve only reads closed under it,
	// while constructService appends closers under it. This lets a service be resolved
	// through a child scope while the Container it is registered with is closed
	// concurrently — the closer is either collected by Close or, if closed is already
	// set, closed by constructService itself.
	closeMu sync.Mutex

	closed bool
}

var _ Scope = (*Container)(nil)

// NewContainer creates a new [Container] with the provided options.
//
// Available options:
//   - [WithService] registers a service with a value or constructor function.
//   - [Module] registers a collection of services.
func NewContainer(opts ...ContainerOption) (*Container, error) {
	c := &Container{
		services: make(map[serviceKey][]*service),
		resolved: make(map[*service]*resolveResult),
	}

	err := applyOptions(opts, func(o ContainerOption) error {
		return o.applyContainer(c)
	})
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

func (c *Container) register(s *service) {
	// The services map is allocated lazily so a child scope that registers no services
	// does not pay for it.
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

func (c *Container) root() *Container {
	root := c
	for root.parent != nil {
		root = root.parent
	}
	return root
}

// lookupService returns a service for the given key.
// When multiple services are registered for the same key (type and tag), this is
// an intentional "last registration wins" for single-value resolution.
// This is documented on Container.Resolve; callers wanting a specific service
// should use a distinct tag.
//
// Use [registeredServices] to get all services registered for a key.
func (c *Container) lookupService(key serviceKey) *service {
	for scope := c; scope != nil; scope = scope.parent {
		svcs, ok := scope.services[key]
		if !ok {
			continue
		}

		return svcs[len(svcs)-1]
	}

	return nil
}

// registeredServices returns every service registered for a key in runtime slice
// resolution order: child to ancestors, preserving registration order within each
// container scope.
func (c *Container) registeredServices(key serviceKey) iter.Seq[*service] {
	return func(yield func(*service) bool) {
		for scope := c; scope != nil; scope = scope.parent {
			for _, svc := range scope.services[key] {
				if !yield(svc) {
					return
				}
			}
		}
	}
}

// NewScope creates a new [Container] with a child scope.
//
// Services registered with the parent container will be inherited by the child.
// For services registered with [Scoped], each child container will create an isolated instance
// when the service is resolved.
//
// Additional services can be registered when creating the new scope if needed and they will be isolated from
// the parent and sibling containers.
//
// Available options:
//   - [WithService] registers a service with a value or a function.
//   - [Module] registers a collection of services.
func (c *Container) NewScope(opts ...ContainerOption) (*Container, error) {
	c.closeMu.Lock()
	closed := c.closed
	c.closeMu.Unlock()

	if closed {
		return nil, errors.Wrap(errContainerClosed, "di.Container.NewScope")
	}

	// resolved is allocated lazily on first cache write (see resolveService), so a
	// scope that never resolves a cached service does not pay for the map.
	scope := &Container{
		parent: c,
	}

	err := applyOptions(opts, func(o ContainerOption) error {
		return o.applyContainer(scope)
	})
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

	return c.lookupService(key) != nil
}

// ResolveOption can be used when calling [Resolve], [MustResolve],
// [Container.Resolve], or [Container.Contains].
type ResolveOption interface {
	// applyServiceKey does not use a pointer to optimize allocations.
	applyServiceKey(serviceKey) serviceKey
}

// Resolve a service of the given [reflect.Type].
//
// If more than one service is registered for the type and tag (including no tag),
// resolving it as a single value returns the last-registered service. Resolve the type as
// a slice to get all matching services, or use [WithTag] with a distinct tag to select a
// specific one.
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

	c.closeMu.Lock()
	closed := c.closed
	c.closeMu.Unlock()

	if closed {
		return nil, errors.Wrapf(errContainerClosed, "di.Container.Resolve %s", key)
	}

	// The visitor is created lazily during resolution: a cached result is returned
	// without one, avoiding an allocation on the hot path.
	val, err := resolveKey(ctx, c, key, nil, false)
	if err != nil {
		return val, errors.Wrapf(err, "di.Container.Resolve %s", key)
	}

	return val, nil
}

func resolveKey(
	ctx context.Context,
	scope *Container,
	key serviceKey,
	visitor *resolveVisitor,
	optional bool,
) (any, error) {
	if isUnnamedSliceType(key.Type) {
		return resolveSliceKey(ctx, scope, key, visitor, optional)
	}

	// Look up the service
	svc := scope.lookupService(key)
	if svc == nil {
		// If the service is not found, return an error
		return nil, errServiceNotRegistered
	}

	return resolveService(ctx, scope, key, svc, visitor)
}

func resolveSliceKey(
	ctx context.Context,
	scope *Container,
	key serviceKey,
	visitor *resolveVisitor,
	optional bool,
) (any, error) {
	sliceVal := reflect.MakeSlice(key.Type, 0, 0)
	elemType := key.Type.Elem()
	elemKey := serviceKey{
		Type: elemType,
		Tag:  key.Tag,
	}

	var found bool
	for svc := range scope.registeredServices(elemKey) {
		val, err := resolveService(ctx, scope, elemKey, svc, visitor)
		if err != nil {
			return nil, err
		}

		sliceVal = reflect.Append(sliceVal, safeReflectValue(elemType, val))
		found = true
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
	visitor *resolveVisitor,
) (any, error) {
	if svc.IsValue() {
		// Value services don't need to be resolved--just return the value directly.
		return svc.Value(), nil
	}

	// Check context for errors
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// For singleton services, use the scope the service is registered with.
	// Otherwise, use the current scope.
	lifetime := svc.Lifetime()
	if lifetime == Singleton {
		scope = svc.Scope()
	} else if lifetime == Scoped && scope == svc.Scope() {
		return nil, errors.New("scoped service must be resolved from a child scope")
	}

	if lifetime == Transient {
		// Transient services are not cached; they always construct a new value.
		// Enter guards against a transient service depending on itself.
		visitor = visitor.orNew()
		if !visitor.Enter(svc) {
			return nil, errDependencyCycle
		}
		defer visitor.Leave()

		return constructService(ctx, scope, key, svc, visitor)
	}

	// For Singleton or Scoped services, we cache the result.
resolveAndCache:

	scope.resolvedMu.Lock()
	res, exists := scope.resolved[svc]
	if exists {
		scope.resolvedMu.Unlock()

		if !res.isDone() {
			// The service is being resolved by another goroutine, or by this one
			// further up the stack (a dependency cycle). waitForResult detects the
			// latter via the in-flight result's owner, so a cached hit needs no visitor.
			visitor = visitor.orNew()
			if err := scope.waitForResult(ctx, visitor, res, key); err != nil {
				return nil, err
			}
		}

		if res.failed {
			// The service was not cached due to a non-cacheable error, so we need to retry.
			goto resolveAndCache
		}

		// Return the cached result
		return res.val, res.err
	}

	// This goroutine will resolve the service.
	// Enter before publishing the in-flight result so a cycle back to this service
	// is reported rather than silently re-owned.
	visitor = visitor.orNew()
	if !visitor.Enter(svc) {
		scope.resolvedMu.Unlock()
		return nil, errDependencyCycle
	}
	defer visitor.Leave()

	res = newResolveResult(visitor)
	if scope.resolved == nil {
		scope.resolved = make(map[*service]*resolveResult)
	}
	scope.resolved[svc] = res
	scope.resolvedMu.Unlock()

	cached := false
	defer func() {
		if !cached {
			// The error is not cacheable, or the constructor function panicked.
			// Remove the slot BEFORE releasing waiters so no one serves this result,
			// and signal them to re-resolve rather than handing them val/err.
			scope.resolvedMu.Lock()
			delete(scope.resolved, svc)
			scope.resolvedMu.Unlock()

			res.failed = true
		}

		close(res.done)
	}()

	val, err := constructService(ctx, scope, key, svc, visitor)

	if err == nil || svc.IsErrorCacheable(ctx, err) {
		res.val, res.err = val, err
		cached = true
	}

	return val, err
}

func constructService(
	ctx context.Context,
	scope *Container,
	key serviceKey,
	svc *service,
	visitor *resolveVisitor,
) (val any, err error) {
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

	// Create the service
	val, err = svc.New(depVals)

	// Skip the rest if there was an error
	if err != nil {
		return val, err
	}

	// Add Closer for the service
	if closer := svc.CloserFor(val); closer != nil {
		scope.closeMu.Lock()
		if scope.closed {
			scope.closeMu.Unlock()

			// The scope was closed while the service was being constructed.
			// This can happen when a service is resolved through a child scope
			// while the Container the service is registered with is closed concurrently.
			// The Container can no longer close the service, so close it now
			// and fail the resolution.
			closeErr := closer.Close(ctx)
			return nil, errors.Join(errContainerClosed, closeErr)
		}
		scope.closers = append(scope.closers, closer)
		scope.closeMu.Unlock()
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
	// Mark the Container closed and take the closers under closedMu.
	// A service registered with this Container can be resolved through a child scope
	// concurrently with Close. Its closer is either collected here,
	// or constructService sees closed and closes the service itself.
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return errors.Wrap(errContainerClosed, "di.Container.Close: closed already")
	}
	c.closed = true
	closers := c.closers
	c.closers = nil
	c.closeMu.Unlock()

	// Close services in LIFO order
	// This is important because of dependencies
	var errs []error
	for i := len(closers) - 1; i >= 0; i-- {
		err := closers[i].Close(ctx)
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

func newResolveResult(owner *resolveVisitor) *resolveResult {
	return &resolveResult{
		done:  make(chan struct{}),
		owner: owner,
	}
}

type resolveResult struct {
	done chan struct{}
	val  any
	err  error

	// owner is the resolveVisitor of the goroutine resolving this result.
	// It is set before the result is published to the resolved map and never changes.
	owner *resolveVisitor

	failed bool
}

// isDone returns true if the result has been resolved and Done has been closed.
func (r *resolveResult) isDone() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}

// waitForResult blocks until res is resolved by another goroutine, or ctx is done.
//
// It registers the wait so cycles of goroutines waiting on each other are detected
// (and reported as dependency cycle errors) instead of deadlocking.
func (c *Container) waitForResult(ctx context.Context, visitor *resolveVisitor, res *resolveResult, key serviceKey) error {
	root := c.root()

	if err := root.beginWaiting(visitor, res, key); err != nil {
		return err
	}

	var err error
	select {
	case <-res.done:
	case <-ctx.Done():
		err = ctx.Err()
	}

	root.waitingMu.Lock()
	visitor.blockedOn = nil
	root.waitingMu.Unlock()

	return err
}

// beginWaiting registers that this resolution is about to block until res is resolved
// by another goroutine. It must be called on the root Container.
//
// It returns an error if blocking would deadlock:
// when services with a dependency cycle are resolved concurrently, the goroutines can
// each end up owning one in-flight result while waiting on another, forming a cycle of
// goroutines waiting on each other. The per-goroutine visitor cannot see such a cycle,
// so we follow the chain of blocked resolutions here. If it leads back to this visitor,
// waiting would never return, and we report a dependency cycle instead.
func (c *Container) beginWaiting(visitor *resolveVisitor, res *resolveResult, key serviceKey) error {
	c.waitingMu.Lock()
	defer c.waitingMu.Unlock()

	depKeys := make([]serviceKey, 0, 8)
	seen := make(map[*resolveResult]struct{})

	for cur := res; ; {
		if cur.isDone() {
			// The result is resolved, so anyone still waiting on it will wake up.
			// The chain is not blocked.
			break
		}
		if cur.owner == visitor {
			// The chain of waiting goroutines leads back to us:
			// wrap the error with the dependency chain the same way nested
			// resolve calls would report it.
			err := errDependencyCycle
			for i := len(depKeys) - 1; i >= 0; i-- {
				err = errors.Wrapf(err, "dependency %s", depKeys[i])
			}
			return err
		}
		if _, ok := seen[cur]; ok {
			// Defensive: don't follow a foreign cycle forever.
			break
		}
		seen[cur] = struct{}{}

		next := cur.owner.blockedOn
		if next == nil {
			// The owning goroutine is running and will make progress.
			// If it blocks later, it will see our edge and run this same check.
			break
		}
		depKeys = append(depKeys, cur.owner.blockedOnKey)
		cur = next
	}

	visitor.blockedOn = res
	visitor.blockedOnKey = key
	return nil
}

const visitorBufSize = 8

// resolveVisitor tracks the services being resolved by a single resolve call
// to detect dependency cycles.
//
// It must always be used as a pointer: it serves as the identity of the resolve call
// for deadlock detection, and entered is backed by the inline buf array.
type resolveVisitor struct {
	// blockedOnKey is the service key this resolution was resolving when it blocked.
	// It is guarded by the root Container's waitingMu, together with blockedOn.
	blockedOnKey serviceKey

	// buf is the initial backing array for entered.
	// append spills to the heap if a dependency chain is deeper.
	buf [visitorBufSize]*service

	// blockedOn is the in-flight result this resolution is waiting on, if any.
	blockedOn *resolveResult

	// entered is a stack of the services currently being resolved.
	// Enter and Leave calls are strictly nested, and dependency chains are shallow.
	entered []*service
}

func newResolveVisitor() *resolveVisitor {
	v := &resolveVisitor{}
	v.entered = v.buf[:0]
	return v
}

// orNew returns v, or a new visitor if v is nil.
// This lets a resolve start without allocating a visitor and create one only if
// resolution reaches a path that needs cycle or deadlock detection.
func (v *resolveVisitor) orNew() *resolveVisitor {
	if v != nil {
		return v
	}
	return newResolveVisitor()
}

func (v *resolveVisitor) Enter(s *service) bool {
	if slices.Contains(v.entered, s) {
		return false
	}

	v.entered = append(v.entered, s)
	return true
}

// Leave pops the last entered service.
func (v *resolveVisitor) Leave() {
	v.entered = v.entered[:len(v.entered)-1]
}
