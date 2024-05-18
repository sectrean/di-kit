package di

import (
	"reflect"

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

// WithService registers the given function or value with the container.
//
// The fnOrValue argument must be a function or a value.
// The function may take any number of arguments. These dependencies must be registered with the container.
// The function may also accept a [context.Context].
// The function must return a service and optionally an error.
func WithService(fnOrValue any, opts ...ServiceOption) ContainerOption {
	return containerOptionFunc(func(c *Container) error {
		if _, ok := fnOrValue.(ServiceOption); ok {
			return errors.New("with service: unexpected ServiceOption for first arg")
		}

		t := reflect.TypeOf(fnOrValue)

		var svc service
		var err error

		switch t.Kind() {
		case reflect.Func:
			svc, err = newFuncService(fnOrValue, opts...)
		case reflect.Interface, reflect.Ptr, reflect.Struct:
			svc, err = newValueService(fnOrValue, opts...)
		default:
			err = errors.Errorf("unsupported kind %v", t.Kind())
		}

		if err != nil {
			return errors.Wrapf(err, "with service %T", fnOrValue)
		}

		c.register(svc)
		return nil
	})
}

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
			return errors.Wrap(ErrContainerClosed, "with parent")
		}

		c.parent = parent
		return nil
	})
}
