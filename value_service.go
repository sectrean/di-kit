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

func newValueService(val any, opts ...ServiceOption) (*valueService, error) {
	t := reflect.TypeOf(val)
	v := reflect.ValueOf(val)

	if err := validateServiceType(t); err != nil {
		return nil, err
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

func (s *valueService) addAlias(alias reflect.Type) error {
	if !s.t.AssignableTo(alias) {
		return errors.Errorf("type %s not assignable to %s", s.t, alias)
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

func (s *valueService) AsCloser(val any) Closer {
	// The container is not responsible for closing this value by default.
	// But if a closer factory is provided, use it.
	if val != nil && s.closerFactory != nil {
		return s.closerFactory(val)
	}

	return nil
}

func (s *valueService) setCloserFactory(cf closerFactory) {
	s.closerFactory = cf
}

func (s *valueService) New(deps []reflect.Value) (any, error) {
	return s.val, nil
}

var _ service = &valueService{}
