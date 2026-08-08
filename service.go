package di

import (
	"context"
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
// (It will be registered as the actual type even if the variable was declared as an interface.)
//
// A service can be almost any named type including structs, interfaces, basic types, functions, or a pointer to a named type.
// A function with a named type will be treated as a value service, not a constructor function.
//
// The following types cannot be registered as services:
//   - The reserved types [error], [context.Context], and [Scope].
//   - Unnamed types (such as slices, maps, and function types) and built-in types like int.
//   - Any type declared in the di package (such as a [ServiceOption]),
//     to guard against accidentally registering an option like [Singleton].
//
// A function service may depend on any type that can be registered as a service, plus:
//   - [context.Context] and [Scope], which are provided by the Container rather than resolved.
//   - A slice of a registered type (such as []Service), which resolves to all services
//     registered for the element type.
//
// Available options:
//   - [Lifetime] is used to specify how services are created when resolved.
//   - [As] overrides the type a service is registered as.
//   - [WithTag] specifies a tag differentiate between services of the same type.
//   - [WithTagged] specifies a tag for a service dependency.
//   - [UseCloseFunc] specifies a function to be called when the service is closed.
//   - [IgnoreCloser] specifies that the service should not be closed by the Container.
//     Function services are closed by default if they implement [Closer] or a compatible function signature.
//   - [UseCloser] specifies that the service should be closed by the Container if it implements [Closer] or a compatible function signature.
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

	return containerOption(func(c *Container) error {
		v := reflect.ValueOf(funcOrValue)
		if isNil(v) {
			return errors.New("di.WithService: funcOrValue is nil")
		}

		s, err := newService(c, v, opts...)
		if err != nil {
			return errors.Wrapf(err, "di.WithService %s", v.Type())
		}

		c.register(s)
		return nil
	})
}

// ServiceOption is used to configure service registration when calling [WithService].
type ServiceOption interface {
	applyService(*service) error
}

type serviceOption func(*service) error

func (o serviceOption) applyService(s *service) error {
	return o(s)
}

// As registers the service as type Service when calling [WithService].
//
// By default, function services will be registered as the constructor function return type.
// Value services will be registered as the actual type of the value.
//
// Use [As] to register the service as an implemented interface.
// This will override the default registration behavior.
// The original type can also be registered using [As].
//
// This option will return an error if the service type is not assignable to type Service.
func As[Service any]() ServiceOption {
	return serviceOption(func(s *service) error {
		t := reflect.TypeFor[Service]()

		if ok := validateServiceType(t); !ok {
			return errors.Errorf("di.As %s: invalid service type", t)
		}
		if !s.Type().AssignableTo(t) {
			return errors.Errorf("di.As %s: type %s not assignable to %s", t, s.Type(), t)
		}

		s.assignables = append(s.assignables, t)
		return nil
	})
}

type serviceKey struct {
	Type reflect.Type
	Tag  any
}

func (k serviceKey) String() string {
	if k.Tag == nil {
		return k.Type.String()
	}
	return fmt.Sprintf("%s: WithTag %v", k.Type, k.Tag)
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

	// We don't want someone to accidentally register a ServiceOption or something.
	if t.PkgPath() == typeScope.PkgPath() {
		return false
	}

	// Don't allow registering unnamed basic types as services.
	return t.PkgPath() != "" && t.Name() != ""
}

func validateDependencyType(t reflect.Type) bool {
	switch t {
	// These special types are provided by the Container rather than resolved.
	case typeContext,
		typeScope:
		return true
	}

	if isUnnamedSliceType(t) {
		t = t.Elem()
	}

	return validateServiceType(t)
}

type closerFactory = func(any) Closer

type service struct {
	scope         *Container
	v             reflect.Value
	t             reflect.Type
	deps          []serviceKey
	tags          []any
	closerFactory closerFactory
	assignables   []reflect.Type
	lifetime      Lifetime
}

func newService(c *Container, v reflect.Value, opts ...ServiceOption) (*service, error) {
	s := &service{
		scope:    c,
		v:        v,
		lifetime: Singleton,
	}
	var err error

	if !s.IsValue() {
		// Func service
		err = s.initFuncService(v.Type())
	} else {
		// Value service
		err = s.initValueService(v.Type())
	}
	if err != nil {
		return nil, err
	}

	if err := s.apply(opts...); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *service) apply(opts ...ServiceOption) error {
	return applyOptions(opts, func(o ServiceOption) error { return o.applyService(s) })
}

func (s *service) initFuncService(funcType reflect.Type) error {
	// Figure out the service type
	switch {
	case funcType.NumOut() == 1:
		s.t = funcType.Out(0)
	case funcType.NumOut() == 2 && funcType.Out(1) == typeError:
		s.t = funcType.Out(0)
	default:
		return errors.New("function must return Service or (Service, error)")
	}

	if ok := validateServiceType(s.t); !ok {
		return errors.New("invalid service type")
	}

	// Get the dependencies and validate dependency types
	var errs []error

	if funcType.NumIn() > 0 {
		s.deps = make([]serviceKey, funcType.NumIn())
		for i := range funcType.NumIn() {
			depType := funcType.In(i)

			if ok := validateDependencyType(depType); !ok {
				err := errors.Errorf("invalid dependency type %s", depType)
				errs = append(errs, err)
				continue
			}

			s.deps[i] = serviceKey{
				Type: depType,
			}
		}
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}

	s.closerFactory = getCloser

	return nil
}

func (s *service) initValueService(valType reflect.Type) error {
	if ok := validateServiceType(valType); !ok {
		return errors.New("invalid service type")
	}

	s.t = valType
	return nil
}

func (s *service) Scope() *Container { return s.scope }

// Type of the service. This is the return type of the constructor function or the actual type of the value.
func (s *service) Type() reflect.Type { return s.t }

// Named function types are treated as values, not constructor functions.
func (s *service) isFunc() bool                { return s.v.Kind() == reflect.Func && s.v.Type().Name() == "" }
func (s *service) IsValue() bool               { return !s.isFunc() }
func (s *service) Lifetime() Lifetime          { return s.lifetime }
func (s *service) Dependencies() []serviceKey  { return s.deps }
func (s *service) Tags() []any                 { return s.tags }
func (s *service) Assignables() []reflect.Type { return s.assignables }

func (s *service) Value() any {
	return s.v.Interface()
}

func (s *service) Func() reflect.Value {
	return s.v
}

func (s *service) New(deps []reflect.Value) (val any, err error) {
	// Call the function
	var out []reflect.Value
	if s.Func().Type().IsVariadic() {
		out = s.Func().CallSlice(deps)
	} else {
		out = s.Func().Call(deps)
	}

	// Get the return value and error, if any
	if !isNil(out[0]) {
		val = out[0].Interface()
	}
	if len(out) == 2 && !isNil(out[1]) {
		err = out[1].Interface().(error)
	}

	return val, err
}

func (s *service) CloserFor(val any) Closer {
	if val == nil {
		return nil
	}

	if s.closerFactory != nil {
		return s.closerFactory(val)
	}

	return nil
}

func (s *service) IsErrorCacheable(ctx context.Context, err error) bool {
	if errors.Is(err, errDependencyCycle) {
		return false // property of the resolution chain
	}
	if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
		return false // property of the caller's context
	}
	return true
}

func (s *service) String() string {
	return s.v.Type().String()
}
