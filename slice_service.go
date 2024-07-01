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

func (s *sliceService) Key() serviceKey {
	return serviceKey{
		Type: s.t,
	}
}

func (s *sliceService) Lifetime() Lifetime {
	return Transient
}

func (s *sliceService) Dependencies() []serviceKey {
	return s.deps
}

func (s *sliceService) AsCloser(val any) Closer {
	// Closers for the individual services will be added to the container
	// as they are resolved.
	return nil
}

func (s *sliceService) New(deps []reflect.Value) (any, error) {
	sliceType := reflect.SliceOf(s.t)
	slice := reflect.MakeSlice(sliceType, 0, len(deps))
	slice = reflect.Append(slice, deps...)

	return slice.Interface(), nil
}

func (s *sliceService) NextItemKey() serviceKey {
	index := len(s.deps)
	key := serviceKey{
		Type: s.t,
		Tag:  sliceItemTag(index),
	}

	s.deps = append(s.deps, key)

	return key
}

type sliceItemTag int
