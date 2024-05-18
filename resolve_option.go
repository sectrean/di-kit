package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// ResolveOption can be used when calling [Container.Resolve] and [Resolve].
//
// Available options:
//   - [WithTag]
type ResolveOption interface {
	applyResolveConfig(*resolveConfig) error
}

type resolveConfig struct {
	t   reflect.Type
	tag any
}

func newResolveConfig(t reflect.Type, opts []ResolveOption) (resolveConfig, error) {
	config := resolveConfig{
		t: t,
	}

	var errs errors.MultiError
	for _, opt := range opts {
		err := opt.applyResolveConfig(&config)
		errs = errs.Append(err)
	}

	return config, errs.Join()
}

func (c resolveConfig) serviceKey() serviceKey {
	return serviceKey{
		Type: c.t,
		Tag:  c.tag,
	}
}
