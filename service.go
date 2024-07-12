package di

import (
	"fmt"
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// WithService registers the provided function or value with a new Container
// when calling [NewContainer] or [Container.NewScope].
//
// If a function is provided, it will be called to create the service when resolved.
//
// This function can take any number of arguments which will also be resolved from the Container.
// The function may also accept a [context.Context] or [di.Scope].
//
// The function must return a service, or the service and an error.
// The service will be registered as the return type of the function (struct, pointer, or interface).
//
// If the function returns an error, this error will be returned when the service is resolved, either directly or as a dependency.
// If the function returns nil for the service, it will not be treated as an error.
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
//   - [WithTag] specifies a tag differentiate between services of the same type.
//   - [WithTagged] specifies a tag for a service dependency.
//   - [WithCloseFunc] specifies a function to be called when the service is closed.
//   - [IgnoreClose] specifies that the service should not be closed by the Container.
//     Function services are closed by default if they implement [Closer] or a compatible function signature.
//   - [WithClose] specifies that the service should be closed by the Container if it implements [Closer] or a compatible function signature.
//     This is the default for function services. Value services will not be closed by default.
func WithService(funcOrValue any, opts ...ServiceOption) ContainerOption {
	// Use a single WithService function for both function and value services
	// because it's easier to use than separate functions.
	//
	// Examples:
	// WithFunc(NewService) // Correct
	// WithFunc(NewService()) // Wrong - easy mistake
	// WithValue(NewService()) // Correct
	// WithValue(NewService) // Wrong - easy mistake
	// WithService(NewService) // This works as a func
	// WithService(NewService()) // This works as a value

	return newContainerOption(orderService, func(c *Container) error {
		if funcOrValue == nil {
			return errors.Errorf("with service: funcOrValue is nil")
		}

		if _, ok := funcOrValue.(ServiceOption); ok {
			return errors.Errorf("with service %T: unexpected ServiceOption as funcOrValue", funcOrValue)
		}

		t := reflect.TypeOf(funcOrValue)

		var sr serviceRegistration
		var err error
		if t.Kind() == reflect.Func {
			sr, err = newFuncService(c, funcOrValue, opts...)
		} else {
			sr, err = newValueService(c, funcOrValue, opts...)
		}

		if err != nil {
			return errors.Wrapf(err, "with service %T", funcOrValue)
		}

		c.register(sr)
		return nil
	})
}

func validateServiceType(t reflect.Type) error {
	switch t {
	// These are the only special types used by the Container.
	case contextType,
		scopeType,
		errorType:
		return errors.New("invalid service type")
	}

	switch t.Kind() {
	case reflect.Interface,
		reflect.Ptr,
		reflect.Struct:
		return nil
	}

	return errors.New("invalid service type")
}

// ServiceOption is used to configure service registration calling [WithService].
type ServiceOption interface {
	applyService(serviceRegistration) error
}

type serviceOption func(serviceRegistration) error

func (o serviceOption) applyService(sr serviceRegistration) error {
	return o(sr)
}

// service provides information about a service and how to resolve it.
type service interface {
	// Key returns the key of the service.
	Key() serviceKey

	// Scope is the Container that the service is registered with.
	Scope() *Container

	// Lifetime returns the lifetime of the service.
	Lifetime() Lifetime

	// Dependencies returns the types of the services that this service depends on.
	Dependencies() []serviceKey

	// New uses the dependencies to create a new instance of the service.
	New(deps []reflect.Value) (any, error)

	// CloserFor returns a Closer for the service.
	CloserFor(val any) Closer
}

type serviceRegistration interface {
	service

	// Type returns the type of the service.
	Type() reflect.Type

	// Tag returns the tag of the service.
	Tag() any
	SetTag(any)

	// Lifetime returns the lifetime of the service.
	Lifetime() Lifetime
	SetLifetime(Lifetime)

	// Aliases returns the types that this service can be resolved as.
	Aliases() []reflect.Type
	AddAlias(reflect.Type) error

	SetCloserFactory(closerFactory)
}

type serviceKey struct {
	Type reflect.Type
	Tag  any
}

func (k serviceKey) String() string {
	if k.Tag == nil {
		return k.Type.String()
	}
	return fmt.Sprintf("%s (Tag %v)", k.Type, k.Tag)
}

// type resolvedService interface {
// 	Result() (any, error)
// }

// type valueResult struct {
// 	val any
// }

// func (r valueResult) Result() (any, error) {
// 	return r.val, nil
// }

// type servicePromise struct {
// 	val  any
// 	err  error
// 	done chan struct{}
// }

// func newServicePromise() (*servicePromise, func(any, error)) {
// 	p := &servicePromise{
// 		done: make(chan struct{}),
// 	}

// 	return p, p.resolve
// }

// func (p *servicePromise) resolve(val any, err error) {
// 	p.val = val
// 	p.err = err

// 	close(p.done)
// }

// func (p *servicePromise) Result() (any, error) {
// 	// Block until val and err have been set
// 	<-p.done

// 	return p.val, p.err
// }
