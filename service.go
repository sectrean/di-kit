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
	setLifetime(Lifetime)

	// Aliases returns the types that this service can be resolved as.
	Aliases() []reflect.Type
	addAlias(reflect.Type) error

	// Key returns the key of the service.
	Key() any
	setKey(any)

	// Dependencies returns the types of the services that this service depends on.
	Dependencies() []serviceKey

	// New uses the dependencies to create a new instance of the service.
	New(deps []reflect.Value) (any, error)

	// AsCloser returns a Closer for the service.
	AsCloser(val any) Closer
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

type resolvedService interface {
	Result() (any, error)
}

type valueResult struct {
	val any
}

func (r valueResult) Result() (any, error) {
	return r.val, nil
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

func (p *servicePromise) setResult(val any, err error) {
	p.val = val
	p.err = err
	close(p.done)
}

func (p *servicePromise) Result() (any, error) {
	<-p.done
	return p.val, p.err
}
