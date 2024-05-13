package di

import (
	"github.com/johnrutherford/di-kit/internal/errors"
)

// TagOption is used to specify the tag associated with a service.
type TagOption interface {
	RegisterFuncOption
	RegisterValueOption
	ResolveOption
	ContainsOption
}

// WithTag is used to specify the tag associated with a service.
//
// WithTag can be used when registering a service, resolving a service,
// or checking if a service is contained within a Scope.
func WithTag(tag any) TagOption {
	return tagOption{tag}
}

// WithDependencyTag is used to specify a tag for a dependency.
func WithDependencyTag[T any](tag any) RegisterFuncOption {
	t := TypeOf[T]()

	return registerFuncOptionFunc(func(c *funcService) error {
		for _, dep := range c.deps {
			if dep.Type == t {
				dep.Tag = tag
				return nil
			}
		}
		return errors.Errorf("dependency %s not found", t)
	})
}

type tagOption struct {
	tag any
}

func (o tagOption) applyFuncService(s *funcService) error {
	s.tag = o.tag
	return nil
}

func (o tagOption) applyValueService(s *valueService) error {
	s.tag = o.tag
	return nil
}

func (o tagOption) applyResolveConfig(c *resolveConfig) error {
	c.tag = o.tag
	return nil
}

func (o tagOption) applyContainsConfig(c *containsConfig) {
	c.tag = o.tag
}

var _ TagOption = tagOption{}
