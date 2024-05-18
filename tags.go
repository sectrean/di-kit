package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// TagOption is used to specify the tag associated with a service.
//
// See implementation [WithTag].
type TagOption interface {
	RegisterFuncOption
	RegisterValueOption
	ResolveOption
	ContainsOption
}

// WithTag is used to specify the tag associated with a service.
//
// WithTag can be used with:
//   - [RegisterFunc]
//   - [RegisterValue]
//   - [Container.Resolve]
//   - [Container.Contains]
//   - [Resolve]
//   - [MustResolve]
func WithTag(tag any) TagOption {
	return tagOption{tag}
}

// TODO: Use dependency tag with Invoke

// WithDependencyTag is used to specify a tag for a dependency when calling [RegisterFunc].
//
// Example:
//
//	c, err := di.NewContainer(
//		di.RegisterFunc(NewUsersDB, di.WithTag("users")),
//		di.RegisterFunc(NewOrdersDB, di.WithTag("orders")),
//		di.RegisterFunc(NewUsersStore,
//			di.WithDependencyTag[*sql.DB]("users")
//		),
//		di.RegisterFunc(NewOrdersStore,
//			di.WithDependencyTag[*sql.DB]("orders")
//		),
//	)
func WithDependencyTag[T any](tag any) RegisterFuncOption {
	t := reflect.TypeFor[T]()

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
