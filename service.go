package di

import (
	"fmt"
	"reflect"
)

// service provides information about a service and how to resolve it.
type service interface {
	// Type returns the type of the service.
	Type() reflect.Type
	// Aliases returns the types that this service can be resolved as.
	Aliases() []reflect.Type
	// AddAlias adds an alias for the service.
	// The alias must be assignable to the service type.
	AddAlias(alias reflect.Type) error
	// Tag returns the tag of the service.
	Tag() any
	// Dependencies returns the types of the services that this service depends on.
	Dependencies() []serviceKey
	// GetValue uses the dependencies to get an instance of the service.
	GetValue(deps []any) (any, error)
	// GetCloser returns a Closer that will close the service.
	GetCloser(val any) Closer
}

type serviceKey struct {
	Type reflect.Type
	Tag  any
}

func (k serviceKey) String() string {
	if k.Tag == nil {
		return k.Type.String()
	}
	return fmt.Sprintf("%s (Tag %v)", k.Type, k.Tag)
}
