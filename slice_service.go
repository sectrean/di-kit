package di

import (
	"reflect"
)

type sliceService struct {
	t        reflect.Type
	services []service
}

func newSliceService(t reflect.Type) *sliceService {
	return &sliceService{
		t: t,
	}
}

func (s *sliceService) Type() reflect.Type {
	return s.t
}

func (s *sliceService) Lifetime() Lifetime {
	panic("unimplemented")
}

func (s *sliceService) Aliases() []reflect.Type {
	return nil
}

func (s *sliceService) AddAlias(alias reflect.Type) error {
	panic("unimplemented")
}

func (s *sliceService) Tag() any {
	return nil
}

func (s *sliceService) Dependencies() []serviceKey {
	panic("unimplemented")
}

func (s *sliceService) GetCloser(val any) Closer {
	panic("unimplemented")
}

func (s *sliceService) GetValue(deps []any) (any, error) {
	panic("unimplemented")
}

func (s *sliceService) Add(service service) {
	s.services = append(s.services, service)
}

var _ service = &sliceService{}
