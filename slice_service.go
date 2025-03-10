package di

import (
	"reflect"
)

func newSliceService(t reflect.Type) *sliceService {
	return &sliceService{
		key: serviceKey{
			Type: reflect.SliceOf(t),
		},
	}
}

type sliceService struct {
	key  serviceKey
	deps []serviceKey
}

var _ service = (*sliceService)(nil)

func (s *sliceService) Key() serviceKey {
	return s.key
}

func (s *sliceService) Lifetime() Lifetime {
	// This must be transient because the lifetime of the individual services
	// could be transient or scoped.
	return TransientLifetime
}

func (s *sliceService) Dependencies() []serviceKey {
	return s.deps
}

func (s *sliceService) CloserFor(any) Closer {
	// Closers for the individual services will be added to the container
	// as they are resolved.
	return nil
}

func (s *sliceService) New(deps []reflect.Value) (any, error) {
	slice := reflect.MakeSlice(s.key.Type, 0, len(deps))
	slice = reflect.Append(slice, deps...)

	return slice.Interface(), nil
}

func (s *sliceService) NextItemKey() serviceKey {
	index := len(s.deps)
	key := serviceKey{
		Type: s.key.Type.Elem(),
		Tag:  sliceItemTag(index),
	}

	s.deps = append(s.deps, key)

	return key
}

type sliceItemTag int
