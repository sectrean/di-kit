package di

import (
	"reflect"

	"github.com/google/uuid"
)

type valueService struct {
	id  uuid.UUID
	t   reflect.Type
	val any
}

func newValueService(val any, options *ProvideOptions) (Service, error) {
	t := reflect.TypeOf(val)

	// TODO: See if any options are needed for value services

	svc := &valueService{
		id:  uuid.New(),
		t:   t,
		val: val,
	}

	return svc, nil
}

// Dependencies implements Service
func (*valueService) Dependencies() []reflect.Type {
	return nil
}

// GetCloser implements Service.
func (*valueService) GetCloser(val any) (Closer, bool) {
	return nil, false
}

// GetValue implements Service
func (s *valueService) GetValue(deps []any) (any, error) {
	// The container should not be responsible for closing this value
	return s.val, nil
}

// ID implements Service
func (s *valueService) ID() uuid.UUID {
	return s.id
}

// Lifetime implements Service
func (*valueService) Lifetime() Lifetime {
	return Singleton
}

// Type implements Service
func (s *valueService) Type() reflect.Type {
	return s.t
}

var _ Service = &valueService{}
