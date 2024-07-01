package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

type valueService struct {
	key           serviceKey
	val           any
	aliases       []reflect.Type
	closerFactory func(any) Closer
}

func newValueService(val any, opts ...ServiceOption) (*valueService, error) {
	t := reflect.TypeOf(val)
	v := reflect.ValueOf(val)

	if err := validateServiceType(t); err != nil {
		return nil, err
	}

	svc := &valueService{
		key: serviceKey{Type: t},
		val: v.Interface(),
	}

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

func (s *valueService) Aliases() []reflect.Type {
	return s.aliases
}

func (s *valueService) AddAlias(alias reflect.Type) error {
	if !s.key.Type.AssignableTo(alias) {
		return errors.Errorf("type %s not assignable to %s", s.key.Type, alias)
	}

	s.aliases = append(s.aliases, alias)
	return nil
}

func (s *valueService) Key() serviceKey {
	return s.key
}

func (s *valueService) Type() reflect.Type {
	return s.key.Type
}

func (s *valueService) Lifetime() Lifetime {
	return Singleton
}

func (s *valueService) SetLifetime(Lifetime) {
	// Values are always singletons.
}

func (s *valueService) Tag() any {
	return s.key.Tag
}

func (s *valueService) SetTag(tag any) {
	s.key.Tag = tag
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

func (s *valueService) SetCloserFactory(cf closerFactory) {
	s.closerFactory = cf
}

func (s *valueService) New(deps []reflect.Value) (any, error) {
	return s.val, nil
}

var _ service = &valueService{}
var _ serviceRegistration = &valueService{}
