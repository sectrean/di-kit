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
	return Transient
}

func (s *sliceService) Aliases() []reflect.Type {
	return nil
}

func (s *sliceService) AddAlias(alias reflect.Type) error {
	panic("not supported")
}

func (s *sliceService) Tag() any {
	return nil
}

func (s *sliceService) Dependencies() []serviceKey {
	return nil
}

func (s *sliceService) GetCloser(val any) Closer {
	// Closers for the individual services will be added to the container
	// as they are resolved.
	return nil
}

func (s *sliceService) GetValue(deps []any) (any, error) {
	// TODO: Implement this
	panic("unimplemented")
}

func (s *sliceService) Add(service service) {
	s.services = append(s.services, service)
}

var _ service = &sliceService{}
