package di

import (
	"reflect"
)

type valueService struct {
	typ reflect.Type
	val any
}

func newValueService(rVal reflect.Value, _ *ServiceOptions) (*valueService, error) {
	svc := &valueService{
		typ: rVal.Type(),
		val: rVal.Interface(),
	}

	return svc, nil
}

func (*valueService) Dependencies() []reflect.Type {
	return nil
}

func (*valueService) GetCloser(val any) Closer {
	// The container should not be responsible for closing this value
	return nil
}

func (s *valueService) GetValue(deps []any) (any, error) {
	return s.val, nil
}

func (s *valueService) Type() reflect.Type {
	return s.typ
}

var _ Service = &valueService{}
