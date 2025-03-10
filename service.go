package di

import (
	"fmt"
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

// WithService registers the provided function or value with a new [Container]
// when calling [NewContainer] or [Container.NewScope].
//
// If a function is provided, it will be called to create the service when resolved.
//
// This function can take any number of parameters which will also be resolved from the Container.
// The function may also accept a [context.Context] or [di.Scope].
//
// The function must return a service, or the service and an error.
// The service will be registered as the return type of the function, which must be an interface,
// a struct, or a pointer to an interface or struct.
//
// If the function returns an error, this error will be returned when the service is resolved,
// either directly or as a dependency.
// If the function returns nil for the service, it will not be treated as an error.
//
// If the resolved service implements [Closer], or a compatible Close method signature,
// it will be closed when the Container is closed.
//
// If a value is provided, it will be returned as the service when resolved.
// The value can be a struct or a pointer to a struct.
// (It will be registered as the actual type even if the variable was declared as an interface.)
//
// Available options:
//   - [Lifetime] is used to specify how services are created when resolved.
//   - [As] overrides the type a service is registered as.
//   - [WithTag] specifies a tag differentiate between services of the same type.
//   - [WithTagged] specifies a tag for a service dependency.
//   - [WithCloseFunc] specifies a function to be called when the service is closed.
//   - [IgnoreClose] specifies that the service should not be closed by the Container.
//     Function services are closed by default if they implement [Closer] or a compatible function signature.
//   - [WithClose] specifies that the service should be closed by the Container if it implements [Closer] or a compatible function signature.
//     This is the default for function services. Value services will not be closed by default.
func WithService(funcOrValue any, opts ...ServiceOption) ContainerOption {
	// Use a single WithService function for both function and value services
	// because it's a better UX.
	// Examples:
	// WithFunc(NewService) // Correct
	// WithFunc(NewService()) // Easy mistake causes runtime error
	// WithValue(NewService()) // Correct
	// WithValue(NewService) // Easy mistake causes runtime error
	// WithService(NewService) // This works as a func
	// WithService(NewService()) // This works as a value

	return newContainerOption(orderService, func(c *Container) error {
		if funcOrValue == nil {
			return errors.Errorf("WithService: funcOrValue is nil")
		}

		t := reflect.TypeOf(funcOrValue)

		var sc serviceConfig
		var err error
		if t.Kind() == reflect.Func {
			sc, err = newFuncService(funcOrValue, opts...)
		} else {
			sc, err = newValueService(funcOrValue, opts...)
		}

		if err != nil {
			return errors.Wrapf(err, "WithService %T", funcOrValue)
		}

		c.register(sc)
		return nil
	})
}

func validateServiceType(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t {
	// These special types cannot be registered as services
	case typeContext,
		typeScope,
		typeError:
		return false
	}

	// We don't want someone to accidentally register a ContainerOption or something.
	if t.PkgPath() == typeScope.PkgPath() {
		return false
	}

	if t.Kind() == reflect.Interface || t.Kind() == reflect.Struct {
		return true
	}

	return false
}

func validateDependencyType(t reflect.Type) bool {
	switch t {
	// These special types are allowed as dependencies
	case typeContext,
		typeScope,
		typeError:
		return true
	}

	if t.Kind() == reflect.Slice {
		t = t.Elem()
	}

	return validateServiceType(t)
}

// As registers the service as type Service when calling [WithService].
//
// By default, function services will be registered as the constructor function return type.
// Value services will be registered as the actual type of the value.
//
// For your services to depend on interfaces, you must provide the implemented interface type(s) when creating the [Container].
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithService(db.NewSQLDB,	// NewSQLDB() *db.SQLDB
//			di.As[db.DB](),	// Register as an implemented interface
//		),
//		di.WithService(storage.NewDBStorage),	// NewDBStorage(db.DB) *storage.DBStorage
//		// ...
//	)
//
// If you use any [As] option, the original type will not be registered unless you specify it with another [As] option.
//
// This option will return an error if the service type is not assignable to type Service.
func As[Service any]() ServiceOption {
	return serviceOption(func(sc serviceConfig) error {
		t := reflect.TypeFor[Service]()
		if !sc.Type().AssignableTo(t) {
			return errors.Errorf("As %s: type %s not assignable to %s", t, sc.Type(), t)
		}

		assignables := append(sc.Assignables(), t)
		sc.SetAssignables(assignables)

		return nil
	})
}

// ServiceOption is used to configure service registration calling [WithService].
type ServiceOption interface {
	applyServiceConfig(serviceConfig) error
}

type serviceOption func(serviceConfig) error

func (o serviceOption) applyServiceConfig(sc serviceConfig) error {
	return o(sc)
}

// service provides information about a service and how to resolve it.
type service interface {
	// Key returns the key of the service.
	Key() serviceKey

	// Lifetime returns the lifetime of the service.
	Lifetime() Lifetime

	// Dependencies returns the types of the services that this service depends on.
	Dependencies() []serviceKey

	// New uses the dependencies to create a new instance of the service.
	New(deps []reflect.Value) (any, error)

	// CloserFor returns a Closer for the service.
	CloserFor(val any) Closer
}

// serviceConfig provides information to register a service with a Container.
type serviceConfig interface {
	service

	// Type returns the type of the service.
	Type() reflect.Type

	// Tag returns the tag of the service.
	Tag() any
	SetTag(any)

	// Lifetime returns the lifetime of the service.
	Lifetime() Lifetime
	SetLifetime(Lifetime) error

	// Assignables returns the types that this service can be resolved as.
	Assignables() []reflect.Type
	SetAssignables([]reflect.Type)

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
