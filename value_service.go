package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

type valueService struct {
	key           serviceKey
	val           any
	scope         *Container
	closerFactory func(any) Closer
	aliases       []reflect.Type
}

func newValueService(scope *Container, val any, opts ...ServiceOption) (*valueService, error) {
	t := reflect.TypeOf(val)
	v := reflect.ValueOf(val)

	if err := validateServiceType(t); err != nil {
		return nil, err
	}

	svc := &valueService{
		scope: scope,
		key:   serviceKey{Type: t},
		val:   v.Interface(),
	}

	err := applyOptions(opts, func(opt ServiceOption) error {
		return opt.applyServiceConfig(svc)
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *valueService) Scope() *Container {
	return s.scope
}

func (s *valueService) Key() serviceKey {
	return s.key
}

func (s *valueService) Type() reflect.Type {
	return s.key.Type
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

func (s *valueService) Lifetime() Lifetime {
	return SingletonLifetime
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

func (s *valueService) CloserFor(val any) Closer {
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

func (s *valueService) New([]reflect.Value) (any, error) {
	return s.val, nil
}

var _ service = (*valueService)(nil)
var _ serviceConfig = (*valueService)(nil)
