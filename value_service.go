package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

type valueService struct {
	t             reflect.Type
	aliases       []reflect.Type
	key           any
	val           any
	closerFactory func(any) Closer
}

func newValueService(val any, opts ...RegisterOption) (*valueService, error) {
	t := reflect.TypeOf(val)
	v := reflect.ValueOf(val)

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

func (s *valueService) Key() any {
	return s.key
}

func (s *valueService) setKey(key any) {
	s.key = key
}

func (*valueService) Dependencies() []serviceKey {
	return nil
}

func (s *valueService) GetCloser(val any) Closer {
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

func (s *valueService) GetValue(deps []reflect.Value) (any, error) {
	return s.val, nil
}

var _ service = &valueService{}
