package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

type valueService struct {
	t             reflect.Type
	aliases       []reflect.Type
	tag           any
	val           any
	closerFactory func(any) Closer
}

func newValueService(val any, opts ...RegisterOption) (*valueService, error) {
	t := reflect.TypeOf(val)
	v := reflect.ValueOf(val)

	switch t.Kind() {
	case reflect.Interface,
		reflect.Pointer,
		reflect.Struct:
		// These are the supported kinds.

	default:
		return nil, errors.Errorf("unsupported kind %s", t.Kind())
	}

	svc := &valueService{
		t:   t,
		val: v.Interface(),
	}

	var errs errors.MultiError
	for _, opt := range opts {
		err := opt.applyService(svc)
		errs = errs.Append(err)
	}

	return svc, errs.Join()
}

func (s *valueService) Aliases() []reflect.Type {
	return s.aliases
}

func (s *valueService) AddAlias(alias reflect.Type) error {
	if !s.t.AssignableTo(alias) {
		return errors.Errorf("service type %s is not assignable to alias type %s", s.t, alias)
	}

	s.aliases = append(s.aliases, alias)
	return nil
}

func (s *valueService) Type() reflect.Type {
	return s.t
}

func (s *valueService) Lifetime() Lifetime {
	return Singleton
}

func (s *valueService) setLifetime(Lifetime) {
	// Values are always singletons.
}

func (s *valueService) Tag() any {
	return s.tag
}

func (s *valueService) setTag(tag any) {
	s.tag = tag
}

func (*valueService) Dependencies() []serviceKey {
	return nil
}

func (s *valueService) GetCloser(val any) Closer {
	if val == nil {
		return nil
	}

	// The container is not responsible for closing this value by default.
	// But if a closer factory is provided, use it.
	if s.closerFactory != nil {
		return s.closerFactory(val)
	}

	return nil
}

func (s *valueService) setCloserFactory(cf closerFactory) {
	s.closerFactory = cf
}

func (s *valueService) GetValue(deps []any) (any, error) {
	return s.val, nil
}

var _ service = &valueService{}
