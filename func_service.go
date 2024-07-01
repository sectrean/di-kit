package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

type funcService struct {
	key           serviceKey
	fn            reflect.Value
	deps          []serviceKey
	lifetime      Lifetime
	aliases       []reflect.Type
	closerFactory func(any) Closer
}

func newFuncService(fn any, opts ...ServiceOption) (*funcService, error) {
	fnType := reflect.TypeOf(fn)
	fnVal := reflect.ValueOf(fn)

	// Get the return type
	var t reflect.Type
	if fnType.NumOut() == 1 {
		t = fnType.Out(0)
	} else if fnType.NumOut() == 2 && fnType.Out(1) == errorType {
		t = fnType.Out(0)
	} else {
		return nil, errors.New("function must return Service or (Service, error)")
	}

	if err := validateServiceType(t); err != nil {
		return nil, err
	}

	// Get the dependencies
	var deps []serviceKey
	if fnType.NumIn() > 0 {
		deps = make([]serviceKey, fnType.NumIn())
		for i := 0; i < fnType.NumIn(); i++ {
			deps[i] = serviceKey{
				Type: fnType.In(i),
			}
		}
	}

	svc := &funcService{
		key:           serviceKey{Type: t},
		fn:            fnVal,
		deps:          deps,
		closerFactory: getCloser,
	}

	// Apply options
	var errs errors.MultiError
	for _, opt := range opts {
		err := opt.applyService(svc)
		errs = errs.Append(err)
	}

	if len(errs) > 0 {
		return nil, errs.Join()
	}

	return svc, nil
}

func (s *funcService) Key() serviceKey {
	return s.key
}

func (s *funcService) Type() reflect.Type {
	return s.key.Type
}

func (s *funcService) Lifetime() Lifetime {
	return s.lifetime
}

func (s *funcService) SetLifetime(l Lifetime) {
	s.lifetime = l
}

func (s *funcService) Aliases() []reflect.Type {
	return s.aliases
}

func (s *funcService) AddAlias(alias reflect.Type) error {
	if !s.key.Type.AssignableTo(alias) {
		return errors.Errorf("type %s not assignable to %s", s.key.Type, alias)
	}

	s.aliases = append(s.aliases, alias)
	return nil
}

func (s *funcService) Tag() any {
	return s.key.Tag
}

func (s *funcService) SetTag(tag any) {
	s.key.Tag = tag
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
	val := out[0].Interface()

	var err error
	if len(out) == 2 {
		err = out[1].Interface().(error)
	}

	return val, err
}

func (s *funcService) AsCloser(val any) Closer {
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

var _ service = &funcService{}
var _ serviceRegistration = &funcService{}
