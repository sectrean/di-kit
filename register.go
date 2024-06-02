package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// Register the provided function or value with the Container when calling [NewContainer].
//
// If a function is provided, it will be called to create the service when resolved.
//
// This function can take any number of arguments which will also be resolved from the Container.
// The function may also accept a [context.Context] or [di.Scope].
//
// The function must return a service, or the service and an error.
// The service will be registered as the return type of the function (struct, pointer, or interface).
//
// If the resolved service implements [Closer], or a compatible Close method signature,
// it will be closed when the Container is closed.
//
// If a value is provided, it will be returned as the service when resolved.
// The value can be a struct or pointer.
// (It will be registered as the actual type even if the the variable was declared as an interface.)
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
//     This is the default for function services. Value services will not be closed by default.
func Register(funcOrValue any, opts ...RegisterOption) ContainerOption {
	// Use a single Register function for both function and value services
	// because it's easier to use than separate functions.
	//
	// Examples:
	// RegisterFunc(NewService) // Correct
	// RegisterFunc(NewService()) // Wrong - easy mistake
	// RegisterValue(NewService()) // Correct
	// RegisterValue(NewService) // Wrong - easy mistake
	// Register(NewService) // This works as a func
	// Register(NewService()) // This works as a value

	return containerOption(func(c *Container) error {
		if funcOrValue == nil {
			return errors.Errorf("register: funcOrValue is nil")
		}

		if _, ok := funcOrValue.(RegisterOption); ok {
			return errors.Errorf("register %T: unexpected RegisterOption for funcOrValue", funcOrValue)
		}

		t := reflect.TypeOf(funcOrValue)

		var svc service
		var err error

		switch t.Kind() {
		case reflect.Func:
			svc, err = newFuncService(funcOrValue, opts...)
		case reflect.Interface, reflect.Ptr, reflect.Struct:
			svc, err = newValueService(funcOrValue, opts...)
		default:
			err = errors.Errorf("unsupported kind %v", t.Kind())
		}

		if err != nil {
			return errors.Wrapf(err, "register %T", funcOrValue)
		}

		c.register(svc)
		return nil
	})
}

// RegisterOption is used to configure a service registration when calling [Register].
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
