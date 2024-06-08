package di

import (
	"fmt"
	"reflect"
)

// service provides information about a service and how to resolve it.
type service interface {
	// Type returns the type of the service.
	Type() reflect.Type
	// Lifetime returns the lifetime of the service.
	Lifetime() Lifetime
	// Aliases returns the types that this service can be resolved as.
	Aliases() []reflect.Type
	// AddAlias adds an alias for the service.
	// The alias must be assignable to the service type.
	AddAlias(alias reflect.Type) error
	// Key returns the key of the service.
	Key() any
	// Dependencies returns the types of the services that this service depends on.
	Dependencies() []serviceKey
	// GetValue uses the dependencies to get an instance of the service.
	GetValue(deps []reflect.Value) (any, error)
	// GetCloser returns a Closer that will close the service.
	GetCloser(val any) Closer

	setLifetime(Lifetime)
	setKey(any)
	setCloserFactory(closerFactory)
}

// ServiceOption can be used when calling [Container.Contains], [Container.Resolve], and [Resolve].
//
// Available options:
//   - [WithKey]
type ServiceOption interface {
	applyServiceKey(serviceKey) serviceKey
}

type serviceKey struct {
	Type reflect.Type
	Key  any
}

func (k serviceKey) String() string {
	if k.Key == nil {
		return k.Type.String()
	}
	return fmt.Sprintf("%s (Key %v)", k.Type, k.Key)
}

type servicePromise struct {
	val  any
	err  error
	done chan struct{}
}

func newServicePromise() *servicePromise {
	return &servicePromise{
		done: make(chan struct{}),
	}
}

func (f *servicePromise) setResult(val any, err error) {
	f.val = val
	f.err = err
	close(f.done)
}

func (f *servicePromise) Result() (any, error) {
	<-f.done
	return f.val, f.err
}
