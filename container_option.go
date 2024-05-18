package di

import (
	"github.com/johnrutherford/di-kit/internal/errors"
)

// ContainerOption is used to configure a new Container.
type ContainerOption interface {
	applyContainer(*Container) error
}

// TODO: Implement additional Container options:
// - Validate dependencies--make sure all types are resolvable, no cycles

type containerOptionFunc func(*Container) error

func (f containerOptionFunc) applyContainer(c *Container) error {
	return f(c)
}

// RegisterFunc registers the given function with a new Container.
//
// The function must return a service, and optionally an error.
// The function may take any number of arguments. These dependencies must also be registered with the container.
// The function may also accept a [context.Context].
func RegisterFunc(fn any, opts ...RegisterFuncOption) ContainerOption {
	return containerOptionFunc(func(c *Container) error {
		if _, ok := fn.(RegisterFuncOption); ok {
			return errors.New("register func: unexpected RegisterFuncOption for first arg")
		}

		fnSvc, err := newFuncService(fn, opts...)
		if err != nil {
			return errors.Wrapf(err, "register func %T", fn)
		}

		c.register(fnSvc)
		return nil
	})
}

// RegisterFuncOption is an option to use when calling [RegisterFunc].
type RegisterFuncOption interface {
	applyFuncService(*funcService) error
}

type registerFuncOptionFunc func(*funcService) error

func (f registerFuncOptionFunc) applyFuncService(s *funcService) error {
	return f(s)
}

// RegisterValue registers the given value with a new Container.
//
// The value must be a supported kind: interface, pointer, or struct.
// The value will be registered as a singleton.
// The value will not be closed by the container.
func RegisterValue(val any, opts ...RegisterValueOption) ContainerOption {
	return containerOptionFunc(func(c *Container) error {
		if _, ok := val.(RegisterValueOption); ok {
			return errors.New("register value: unexpected RegisterValueOption for first arg")
		}

		valSvc, err := newValueService(val, opts...)
		if err != nil {
			return errors.Wrapf(err, "register value %T", val)
		}

		c.register(valSvc)
		return nil
	})
}

// RegisterValueOption is a functional option for configuring a value service.
type RegisterValueOption interface {
	applyValueService(*valueService) error
}

// RegisterOption is used to configure a service.
// This can be used with [RegisterFunc] and [RegisterValue].
type RegisterOption interface {
	RegisterFuncOption
	RegisterValueOption
}

type serviceOption func(service) error

func (o serviceOption) applyFuncService(s *funcService) error {
	return o(s)
}

func (o serviceOption) applyValueService(s *valueService) error {
	return o(s)
}

var _ RegisterOption = serviceOption(nil)

// WithParent can be used to create a new Container with a child scope.
//
// The child Container will inherit all registered services from the parent Container.
// The child Container will use a new scope for resolving [Scoped] services.
//
// Example:
//
//	childScope, err := NewContainer(
//		WithParent(c),
//		WithService(valueForChildContainer),
//	)
func WithParent(parent *Container) ContainerOption {
	return containerOptionFunc(func(c *Container) error {
		if parent.closed.Load() {
			return errors.Wrap(ErrContainerClosed, "parent closed")
		}

		c.parent = parent
		return nil
	})
}
