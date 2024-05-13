package di

import (
	"reflect"
)

type containsConfig struct {
	t   reflect.Type
	tag any
}

// ContainsOption is a functional option for [Scope.Contains].
//
// Available options:
//   - [WithTag] specifies the tag associated with a service.
type ContainsOption interface {
	applyContainsConfig(*containsConfig)
}

func newContainsConfig(t reflect.Type, opts []ContainsOption) *containsConfig {
	config := &containsConfig{
		t: t,
	}

	for _, opt := range opts {
		opt.applyContainsConfig(config)
	}

	return config
}

func (c *containsConfig) serviceKey() serviceKey {
	return serviceKey{
		Type: c.t,
		Tag:  c.tag,
	}
}
