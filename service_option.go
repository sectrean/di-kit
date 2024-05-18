package di

import (
	"reflect"
)

// ServiceOption can be used when calling [WithService].
//
// Available options:
//   - [Lifetime] specifies how services are created when resolved.
//   - [As] registers an alias for a service.
//   - [WithTag] specifies the tag associated with a service.
//   - [WithDependencyTag] specifies a tag for a dependency when calling [WithService].
//   - [WithCloseFunc] specifies a function to be called when the service is closed.
//   - [IgnoreCloser] specifies that the service should not be closed by the Container.
//     Function services are closed by default if they implement [Closer] or a compatible function signature.
//   - [WithCloser] specifies that the service should be closed by the Container if it implements [Closer] or a compatible function signature.
//     This is the default for funtion services. Value services are not be closed by default.
type ServiceOption interface {
	applyService(s service) error
}

type serviceOption func(service) error

func (o serviceOption) applyService(s service) error {
	return o(s)
}

// As registers an alias for a service. Use when calling [WithService].
func As[T any]() ServiceOption {
	return serviceOption(func(s service) error {
		return s.AddAlias(reflect.TypeFor[T]())
	})
}
