package di

import (
	"reflect"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type funcService struct {
	id       uuid.UUID
	t        reflect.Type
	fn       reflect.Value
	lifetime Lifetime
	deps     []reflect.Type
}

// GetCloser implements Service.
func (*funcService) GetCloser(val any) (Closer, bool) {
	return getCloser(val)
}

func newFuncService(fnVal reflect.Value, options *ProvideOptions) (Service, error) {
	fnType := fnVal.Type()

	// TODO: Figure out how to handle variadic arguments
	// Should we just ignore them or treat it as a slice of the type?

	var t reflect.Type
	// func ([any,]) (T[, error])
	if fnType.NumOut() == 1 || fnType.NumOut() == 2 {
		t = fnType.Out(0)

		if fnType.NumOut() == 2 {
			if fnType.Out(1) != tError {
				return nil, errors.Errorf("fn %v return type %v is not error",
					fnType, fnType.Out(1))
			}
		}
	} else {
		return nil, errors.Errorf("fn %v must return (T[, error])", fnType)
	}

	var deps []reflect.Type
	if fnType.NumIn() > 0 {
		deps := make([]reflect.Type, fnType.NumIn())
		for i := 0; i < fnType.NumIn(); i++ {
			deps[i] = fnType.In(i)
		}
	}

	return &funcService{
		id:       uuid.New(),
		t:        t,
		fn:       fnVal,
		lifetime: options.Lifetime,
		deps:     deps,
	}, nil
}

// Dependencies implements Service
func (s *funcService) Dependencies() []reflect.Type {
	return s.deps
}

// GetValue implements Service
func (s *funcService) GetValue(deps []any) (any, error) {
	// Turn our dependencies into reflect.Values
	in := Map(deps, func(dep any) reflect.Value {
		return reflect.ValueOf(dep)
	})

	// Call the function
	out := s.fn.Call(in)

	// Extract the return value and error, if any
	val := out[0].Interface()

	var err error
	if len(out) == 2 {
		err = out[1].Interface().(error)
	}

	return val, err
}

// ID implements Service
func (s *funcService) ID() uuid.UUID {
	return s.id
}

// Lifetime implements Service
func (s *funcService) Lifetime() Lifetime {
	return s.lifetime
}

// Type implements Service
func (s *funcService) Type() reflect.Type {
	return s.t
}

var _ Service = &funcService{}
