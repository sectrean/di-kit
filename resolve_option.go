package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// ResolveOption is a functional option for resolving services.
type ResolveOption interface {
	applyResolveConfig(*resolveConfig) error
}

type resolveConfig struct {
	t   reflect.Type
	tag any
}

func newResolveConfig(t reflect.Type, opts []ResolveOption) (*resolveConfig, error) {
	config := &resolveConfig{
		t: t,
	}

	var multiErr errors.MultiError
	for _, opt := range opts {
		err := opt.applyResolveConfig(config)
		multiErr = multiErr.Append(err)
	}

	return config, multiErr.Wrap("resolve options")
}

func (c *resolveConfig) serviceKey() serviceKey {
	return serviceKey{
		Type: c.t,
		Tag:  c.tag,
	}
}
