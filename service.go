package di

import (
	"fmt"
	"reflect"
)

// Service provides information about a service and how to resolve it.
type Service interface {
	// Type returns the type of the service.
	Type() reflect.Type
	// Dependencies returns the types of the services that this service depends on.
	Dependencies() []reflect.Type
	// GetValue uses the dependencies to get an instance of the service.
	GetValue(deps []any) (any, error)
	// GetCloser returns a Closer that will close the service.
	GetCloser(val any) Closer
}

// NewService creates a new Service from the given function or value.
func NewService(fnOrValue any, opts ...ServiceOption) (Service, error) {
	var svc Service
	var err error

	options := &ServiceOptions{
		typ: reflect.TypeOf(fnOrValue),
	}
	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, fmt.Errorf("registering %T: %w", fnOrValue, err)
		}
	}

	val := reflect.ValueOf(fnOrValue)

	switch val.Kind() {
	case reflect.Func:
		svc, err = newFuncService(val, options)
	case reflect.Interface, reflect.Ptr, reflect.Struct:
		svc, err = newValueService(val, options)
	default:
		err = fmt.Errorf("unsupported service kind %v", val.Kind())
	}

	if err != nil {
		return nil, fmt.Errorf("registering %T: %w", fnOrValue, err)
	}

	return svc, nil
}
