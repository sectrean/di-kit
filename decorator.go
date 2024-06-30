package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// WithDecorator registers a decorator function with a new Container.
func WithDecorator(decorateFunc any, opts ...DecoratorOption) ContainerOption {
	return newContainerOption(orderDecorator, func(c *Container) error {
		if decorateFunc == nil {
			return errors.New("with decorator: decorateFunc is nil")
		}

		if _, ok := decorateFunc.(DecoratorOption); ok {
			return errors.Errorf("with decorator %T: unexpected DecoratorOption as decorateFunc", decorateFunc)
		}

		d, err := newDecorator(decorateFunc, opts)
		if err != nil {
			return errors.Wrapf(err, "with decorator %T", decorateFunc)
		}

		err = c.registerDecorator(d)
		return errors.Wrapf(err, "with decorator %T", decorateFunc)
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
		return nil, errors.Errorf("expected function, got %T", fn)
	}

	// Validate fn has one return value
	if fnType.NumOut() != 1 {
		return nil, errors.Errorf("function must return Service")
	}

	t := fnType.Out(0)
	if err := validateServiceType(t); err != nil {
		return nil, err
	}

	svcInArgs := false
	deps := make([]serviceKey, fnType.NumIn())

	for i := 0; i < fnType.NumIn(); i++ {
		depType := fnType.In(i)
		if depType == t {
			svcInArgs = true
		}

		deps[i] = serviceKey{
			Type: depType,
		}
	}

	if !svcInArgs {
		return nil, errors.Errorf("function must have a Service argument")
	}

	key := serviceKey{
		Type: t,
	}

	d := &decorator{
		key:  key,
		deps: deps,
		fn:   reflect.ValueOf(fn),
	}

	var errs errors.MultiError
	for _, opt := range opts {
		err := opt.applyDecorator(d)
		errs = errs.Append(err)
	}

	if len(errs) > 0 {
		return nil, errs.Join()
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

func (d *decorator) setTag(tag any) error {
	d.key.Tag = tag

	for i, dep := range d.deps {
		if dep.Type == d.key.Type && dep.Tag == nil {
			d.deps[i].Tag = tag
			return nil
		}
	}

	return errors.New("with tag: argument not found")
}

func (d *decorator) Decorate(deps []reflect.Value) any {
	out := d.fn.Call(deps)
	return out[0].Interface()
}

func (d *decorator) String() string {
	return d.fn.Type().String()
}
