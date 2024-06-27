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
//   - [WithKey] specifies a key differentiate between services of the same type.
//   - [WithKeyed] specifies a key for a service dependency.
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

		var svc service
		var err error
		if t.Kind() == reflect.Func {
			svc, err = newFuncService(funcOrValue, opts...)
		} else {
			svc, err = newValueService(funcOrValue, opts...)
		}

		if err != nil {
			return errors.Wrapf(err, "with service %T", funcOrValue)
		}

		c.register(svc)
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
	applyService(service) error
}

type serviceOption func(service) error

func (o serviceOption) applyService(s service) error {
	return o(s)
}

// service provides information about a service and how to resolve it.
type service interface {
	// TODO: Add Key() serviceKey

	// Type returns the type of the service.
	Type() reflect.Type

	// Lifetime returns the lifetime of the service.
	Lifetime() Lifetime
	setLifetime(Lifetime)

	// Aliases returns the types that this service can be resolved as.
	Aliases() []reflect.Type
	addAlias(reflect.Type) error

	// TODO: Rename Key to Tag?
	// Key returns the key of the service.
	Key() any
	setKey(any)

	// Dependencies returns the types of the services that this service depends on.
	Dependencies() []serviceKey

	// New uses the dependencies to create a new instance of the service.
	New(deps []reflect.Value) (any, error)

	// AsCloser returns a Closer for the service.
	AsCloser(val any) Closer
	setCloserFactory(closerFactory)
}

type serviceKey struct {
	Type reflect.Type
	Key  any
}

func (k serviceKey) String() string {
	if k.Key == nil {
		return k.Type.String()
	}
	return fmt.Sprintf("%s (Key %v)", k.Type, k.Key)
}

type resolvedService interface {
	Result() (any, error)
}

type valueResult struct {
	val any
}

func (r valueResult) Result() (any, error) {
	return r.val, nil
}

type servicePromise struct {
	val  any
	err  error
	done chan struct{}
}

func newServicePromise() *servicePromise {
	return &servicePromise{
		done: make(chan struct{}),
	}
}

func (p *servicePromise) setResult(val any, err error) {
	p.val = val
	p.err = err
	close(p.done)
}

func (p *servicePromise) Result() (any, error) {
	<-p.done
	return p.val, p.err
}
