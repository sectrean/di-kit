package di

import (
	"fmt"
	"reflect"
)

type funcService struct {
	typ           reflect.Type
	fn            reflect.Value
	deps          []reflect.Type
	closerFactory func(any) Closer
	ignoreCloser  bool
}

func newFuncService(fnVal reflect.Value, options *ServiceOptions) (*funcService, error) {
	fnType := fnVal.Type()

	// TODO: Figure out how to handle variadic arguments
	// Should we just ignore them or treat it as a slice of the type?

	var typ reflect.Type
	// func ([any,]) (T[, error])
	if fnType.NumOut() == 1 || fnType.NumOut() == 2 {
		typ = fnType.Out(0)

		if fnType.NumOut() == 2 {
			if fnType.Out(1) != typError {
				return nil, fmt.Errorf("fn %v return type %v is not error",
					fnType, fnType.Out(1))
			}
		}
	} else {
		return nil, fmt.Errorf("fn %v must return (T[, error])", fnType)
	}

	var deps []reflect.Type
	if fnType.NumIn() > 0 {
		deps = make([]reflect.Type, fnType.NumIn())
		for i := 0; i < fnType.NumIn(); i++ {
			deps[i] = fnType.In(i)
		}
	}

	return &funcService{
		typ:           typ,
		fn:            fnVal,
		deps:          deps,
		closerFactory: options.closerFactory,
		ignoreCloser:  options.noCloser,
	}, nil
}

func (s *funcService) Type() reflect.Type {
	return s.typ
}

func (s *funcService) Dependencies() []reflect.Type {
	return s.deps
}

func (s *funcService) GetValue(deps []any) (any, error) {
	var in []reflect.Value
	if len(deps) > 0 {
		in = make([]reflect.Value, len(deps))
		for i := 0; i < len(deps); i++ {
			in[i] = reflect.ValueOf(deps[i])
		}
	}

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

func (s *funcService) GetCloser(val any) Closer {
	if s.closerFactory != nil {
		return s.closerFactory(val)
	}

	// Ignore the Closer if the service is configured to not use it
	if s.ignoreCloser {
		return nil
	}

	return getCloser(val)
}

var _ Service = &funcService{}
