package di

import (
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

type valueService struct {
	scope         *Container
	val           any
	t             reflect.Type
	tag           any
	closerFactory func(any) Closer
	assignables   []reflect.Type
}

func newValueService(c *Container, val any, opts ...ServiceOption) (*valueService, error) {
	t := reflect.TypeOf(val)
	v := reflect.ValueOf(val)

	if ok := validateServiceType(t); !ok {
		return nil, errors.New("invalid service type")
	}

	svc := &valueService{
		scope: c,
		val:   safeAnyValue(v),
		t:     t,
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

func (s *valueService) Type() reflect.Type {
	return s.t
}

func (s *valueService) Assignables() []reflect.Type {
	return s.assignables
}

func (s *valueService) SetAssignables(assignables []reflect.Type) {
	s.assignables = assignables
}

func (s *valueService) Lifetime() Lifetime {
	return SingletonLifetime
}

func (s *valueService) SetLifetime(l Lifetime) error {
	// Values are always singletons.
	if l == SingletonLifetime {
		return nil
	}

	return errors.New("invalid lifetime for value service")
}

func (s *valueService) Tag() any {
	return s.tag
}

func (s *valueService) SetTag(tag any) {
	s.tag = tag
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
