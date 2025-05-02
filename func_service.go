package di

import (
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

type funcService struct {
	scope         *Container
	fn            reflect.Value
	t             reflect.Type
	tag           any
	closerFactory func(any) Closer
	deps          []serviceKey
	assignables   []reflect.Type
	lifetime      Lifetime
}

func newFuncService(c *Container, fn any, opts ...ServiceOption) (*funcService, error) {
	fnType := reflect.TypeOf(fn)
	fnVal := reflect.ValueOf(fn)

	// Get the return type
	var t reflect.Type
	switch {
	case fnType.NumOut() == 1:
		t = fnType.Out(0)
	case fnType.NumOut() == 2 && fnType.Out(1) == typeError:
		t = fnType.Out(0)
	default:
		return nil, errors.New("function must return Service or (Service, error)")
	}

	if ok := validateServiceType(t); !ok {
		return nil, errors.New("invalid service type")
	}

	// Get the dependencies and validate dependency types
	var deps []serviceKey
	var errs []error

	if fnType.NumIn() > 0 {
		deps = make([]serviceKey, fnType.NumIn())
		for i := range fnType.NumIn() {
			depType := fnType.In(i)

			if ok := validateDependencyType(depType); !ok {
				err := errors.Errorf("invalid dependency type %s", depType)
				errs = append(errs, err)
				continue
			}

			deps[i] = serviceKey{
				Type: depType,
			}
		}
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	svc := &funcService{
		scope:         c,
		fn:            fnVal,
		t:             t,
		deps:          deps,
		closerFactory: getCloser,
	}

	err := applyOptions(opts, func(opt ServiceOption) error {
		return opt.applyServiceConfig(svc)
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *funcService) Scope() *Container {
	return s.scope
}

func (s *funcService) Type() reflect.Type {
	return s.t
}

func (s *funcService) Lifetime() Lifetime {
	return s.lifetime
}

func (s *funcService) SetLifetime(l Lifetime) error {
	s.lifetime = l
	return nil
}

func (s *funcService) Assignables() []reflect.Type {
	return s.assignables
}

func (s *funcService) SetAssignables(assignables []reflect.Type) {
	s.assignables = assignables
}

func (s *funcService) Tag() any {
	return s.tag
}

func (s *funcService) SetTag(tag any) {
	s.tag = tag
}

func (s *funcService) Dependencies() []serviceKey {
	return s.deps
}

func (s *funcService) New(deps []reflect.Value) (any, error) {
	var out []reflect.Value

	// Call the function
	if s.fn.Type().IsVariadic() {
		out = s.fn.CallSlice(deps)
	} else {
		out = s.fn.Call(deps)
	}

	// Extract the return value and error, if any
	val := safeAnyValue(out[0])

	var err error
	if len(out) == 2 {
		err = out[1].Interface().(error)
	}

	return val, err
}

func (s *funcService) CloserFor(val any) Closer {
	if val == nil {
		return nil
	}

	if s.closerFactory != nil {
		return s.closerFactory(val)
	}

	return nil
}

func (s *funcService) SetCloserFactory(cf closerFactory) {
	s.closerFactory = cf
}

var _ service = (*funcService)(nil)
var _ serviceConfig = (*funcService)(nil)
