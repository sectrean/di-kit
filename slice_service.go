package di

import (
	"reflect"
)

func newSliceService(t reflect.Type) *sliceService {
	return &sliceService{
		t: t,
	}
}

type sliceService struct {
	t    reflect.Type
	deps []serviceKey
}

var _ service = &sliceService{}

func (s *sliceService) Type() reflect.Type {
	return s.t
}

func (s *sliceService) Lifetime() Lifetime {
	return Transient
}

func (s *sliceService) setLifetime(l Lifetime) {
	panic("not supported")
}

func (s *sliceService) Aliases() []reflect.Type {
	return nil
}

func (s *sliceService) AddAlias(alias reflect.Type) error {
	panic("not supported")
}

func (s *sliceService) Key() any {
	return nil
}

func (s *sliceService) setKey(any) {
	panic("not supported")
}

func (s *sliceService) Dependencies() []serviceKey {
	return s.deps
}

func (s *sliceService) GetCloser(val any) Closer {
	// Closers for the individual services will be added to the container
	// as they are resolved.
	return nil
}

func (s *sliceService) setCloserFactory(closerFactory) {
	panic("not supported")
}

func (s *sliceService) GetValue(deps []reflect.Value) (any, error) {
	sliceType := reflect.SliceOf(s.t)
	slice := reflect.MakeSlice(sliceType, 0, len(deps))
	slice = reflect.Append(slice, deps...)

	return slice.Interface(), nil
}

func (s *sliceService) AddNewItem() serviceKey {
	index := len(s.deps)
	key := serviceKey{
		Type: s.t,
		Key:  sliceItemKey(index),
	}

	s.deps = append(s.deps, key)

	return key
}

type sliceItemKey int
