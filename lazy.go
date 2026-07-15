package di

import (
	"context"
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

type Lazy[Service any] struct {
	scope Scope
	tag   any
}

func (l Lazy[Service]) Resolve(ctx context.Context) (Service, error) {
	var t = reflect.TypeFor[Service]()
	var val Service
	var err error

	var anyVal any
	if l.tag != nil {
		anyVal, err = l.scope.Resolve(ctx, t, WithTag(l.tag))
	} else {
		anyVal, err = l.scope.Resolve(ctx, t)
	}

	if err != nil {
		err = errors.Wrapf(err, "di.Lazy[%s].Resolve", t)
	}
	if anyVal != nil {
		val = anyVal.(Service)
	}

	return val, err
}

func (Lazy[Service]) t() reflect.Type {
	return reflect.TypeFor[Service]()
}

func (Lazy[Service]) newLazy(s Scope, tag any) lazy {
	return Lazy[Service]{
		scope: s,
		tag:   tag,
	}
}

func (Lazy[Service]) newLazyPointer(s Scope, tag any) lazy {
	return &Lazy[Service]{
		scope: s,
		tag:   tag,
	}
}

var _ lazy = Lazy[any]{}
var _ lazy = &Lazy[any]{}

// lazy allows us to work with Lazy[Service] types without knowing the concrete type.
type lazy interface {
	t() reflect.Type
	newLazy(s Scope, tag any) lazy
	newLazyPointer(s Scope, tag any) lazy
}

func isLazyType(t reflect.Type) bool {
	return t.Implements(typeLazy)
}

func getLazyServiceType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return reflect.Zero(t).Interface().(lazy).t()
}

// newLazy uses the requested Lazy type to create an initialized Lazy[Service] value
// that can be used to resolve the service from the given Scope.
func newLazy(s Scope, key serviceKey) lazy {
	if key.Type.Kind() == reflect.Pointer {
		l := reflect.Zero(key.Type.Elem()).Interface().(lazy)
		return l.newLazyPointer(s, key.Tag)
	}

	l := reflect.Zero(key.Type).Interface().(lazy)
	return l.newLazy(s, key.Tag)
}
