package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// Register the provided function or value with the container.
//
// The fnOrValue argument must be a function or a value.
// The function may take any number of arguments. These dependencies must be registered with the container.
// The function may also accept a [context.Context].
// The function must return a service and optionally an error.
func Register(fnOrValue any, opts ...RegisterOption) ContainerOption {
	return containerOption(func(c *Container) error {
		if _, ok := fnOrValue.(RegisterOption); ok {
			return errors.New("with service: unexpected RegisterOption for first arg")
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

// RegisterOption can be used when calling [Register].
//
// Available options:
//   - [Lifetime] is used to specify how services are created when resolved.
//   - [As] registers an alias for a service.
//   - [WithTag] specifies the tag associated with a service.
//   - [WithDependencyTag] specifies a tag for a dependency when calling [Register].
//   - [WithCloseFunc] specifies a function to be called when the service is closed.
//   - [IgnoreCloser] specifies that the service should not be closed by the Container.
//     Function services are closed by default if they implement [Closer] or a compatible function signature.
//   - [WithCloser] specifies that the service should be closed by the Container if it implements [Closer] or a compatible function signature.
//     This is the default for funtion services. Value services are not be closed by default.
type RegisterOption interface {
	applyService(s service) error
}

type registerOption func(service) error

func (o registerOption) applyService(s service) error {
	return o(s)
}

// As registers an alias for a service. Use when calling [Register].
func As[T any]() RegisterOption {
	return registerOption(func(s service) error {
		return s.AddAlias(reflect.TypeFor[T]())
	})
}
