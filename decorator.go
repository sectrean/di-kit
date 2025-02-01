package di

import (
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

// WithDecorator registers a decorator function with a new Container.
// A decorator is used to "decorate" or "wrap" a service.
//
// The decorator function must have a parameter for a Service and return Service.
// The function may accept other parameters that will be resolved from the Container.
// No additional return values are allowed.
//
// Decorator functions will be applied when the service is resolved.
//
// It is possible to register multiple decorators for a service.
// The decorators will be applied in the order they were registered.
//
// This will not validate that the service is registered, because it could get registered in a child scope.
// If a decorator is registered for a service, but the service is never registered,
// the decorator will never be used.
func WithDecorator(decorateFunc any, opts ...DecoratorOption) ContainerOption {
	return newContainerOption(orderDecorator, func(c *Container) error {
		if decorateFunc == nil {
			return errors.New("WithDecorator: decorateFunc is nil")
		}

		if c.parent != nil {
			return errors.New("WithDecorator: decorators cannot be registered with a child scope")
		}

		d, err := newDecorator(decorateFunc, opts)
		if err != nil {
			return errors.Wrapf(err, "WithDecorator %T", decorateFunc)
		}

		c.registerDecorator(d)
		return nil
	})
}

// DecoratorOption is an option for registering a decorator function.
//
// See [WithDecorator] for more information.
type DecoratorOption interface {
	applyDecorator(*decorator) error
}

func newDecorator(fn any, opts []DecoratorOption) (*decorator, error) {
	fnType := reflect.TypeOf(fn)

	// Validate fn is a function
	if fnType.Kind() != reflect.Func {
		return nil, errors.New("invalid decorator type")
	}

	if fnType.PkgPath() == typeScope.PkgPath() {
		return nil, errors.New("invalid decorator type")
	}

	// Validate fn has one return value
	if fnType.NumOut() != 1 {
		return nil, errors.New("function must return Service")
	}

	t := fnType.Out(0)
	if ok := validateServiceType(t); !ok {
		return nil, errors.New("invalid service type")
	}

	deps := make([]serviceKey, fnType.NumIn())
	var errs []error
	var svcInDeps bool

	for i := range fnType.NumIn() {
		depType := fnType.In(i)
		deps[i] = serviceKey{
			Type: depType,
		}
		if depType == t {
			svcInDeps = true
			continue
		}

		if ok := validateDependencyType(depType); !ok {
			err := errors.Errorf("invalid dependency type %s", depType)
			errs = append(errs, err)
		}
	}

	if !svcInDeps {
		err := errors.New("function must have a Service parameter")
		errs = append(errs, err)
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	d := &decorator{
		key: serviceKey{
			Type: t,
		},
		deps: deps,
		fn:   reflect.ValueOf(fn),
	}

	err := applyOptions(opts, func(opt DecoratorOption) error {
		return opt.applyDecorator(d)
	})
	if err != nil {
		return nil, err
	}

	return d, nil
}

type decorator struct {
	key  serviceKey
	fn   reflect.Value
	deps []serviceKey
}

func (d *decorator) Key() serviceKey {
	return d.key
}

func (d *decorator) SetTag(tag any) error {
	d.key.Tag = tag

	for i, dep := range d.deps {
		if dep.Type == d.key.Type && dep.Tag == nil {
			d.deps[i].Tag = tag
			return nil
		}
	}

	return errors.New("dependency not found")
}

func (d *decorator) Decorate(deps []reflect.Value) any {
	out := d.fn.Call(deps)
	return out[0].Interface()
}

func (d *decorator) String() string {
	return d.fn.Type().String()
}
