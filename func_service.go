package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

type funcService struct {
	t             reflect.Type
	aliases       []reflect.Type
	fn            reflect.Value
	lifetime      Lifetime
	tag           any
	deps          []serviceKey
	closerFactory func(any) Closer
}

func newFuncService(fn any, opts ...RegisterOption) (*funcService, error) {
	fnType := reflect.TypeOf(fn)
	fnVal := reflect.ValueOf(fn)

	if fnType.Kind() != reflect.Func {
		return nil, errors.Errorf("expected a function, got %v", fnType)
	}

	// Or are they just treated as slices with reflection?
	_ = fnType.IsVariadic()

	// Get the return type
	var t reflect.Type
	if fnType.NumOut() == 1 {
		t = fnType.Out(0)
	} else if fnType.NumOut() == 2 && fnType.Out(1) == errorType {
		t = fnType.Out(0)
	} else {
		return nil, errors.Errorf("function %v must return T or (T, error)", fnType)
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

	funcSvc := &funcService{
		t:             t,
		deps:          deps,
		fn:            fnVal,
		closerFactory: getCloser,
	}

	// Apply options
	var errs errors.MultiError
	for _, opt := range opts {
		err := opt.applyService(funcSvc)
		errs = errs.Append(err)
	}

	return funcSvc, errs.Join()
}

func (s *funcService) Type() reflect.Type {
	return s.t
}

func (s *funcService) Lifetime() Lifetime {
	return s.lifetime
}

func (s *funcService) setLifetime(l Lifetime) {
	s.lifetime = l
}

func (s *funcService) Aliases() []reflect.Type {
	return s.aliases
}

func (s *funcService) AddAlias(alias reflect.Type) error {
	if !s.t.AssignableTo(alias) {
		return errors.Errorf("service type %s is not assignable to alias type %s", s.t, alias)
	}

	s.aliases = append(s.aliases, alias)
	return nil
}

func (s *funcService) Tag() any {
	return s.tag
}

func (s *funcService) setTag(tag any) {
	s.tag = tag
}

func (s *funcService) Dependencies() []serviceKey {
	return s.deps
}

func (s *funcService) GetValue(deps []reflect.Value) (any, error) {
	// Call the function
	out := s.fn.Call(deps)

	// Extract the return value and error, if any
	val := out[0].Interface()

	var err error
	if len(out) == 2 {
		err = out[1].Interface().(error)
	}

	return val, err
}

func (s *funcService) GetCloser(val any) Closer {
	if val == nil {
		return nil
	}

	if s.closerFactory != nil {
		return s.closerFactory(val)
	}

	return nil
}

func (s *funcService) setCloserFactory(cf closerFactory) {
	s.closerFactory = cf
}

var _ service = &funcService{}
