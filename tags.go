package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// TagOption is used to specify the tag associated with a service.
//
// See implementation [WithTag].
type TagOption interface {
	ServiceOption
	ResolveOption
	ContainsOption
}

// WithTag is used to specify the tag associated with a service.
//
// WithTag can be used with:
//   - [WithService]
//   - [Resolve]
//   - [MustResolve]
//   - [Container.Resolve]
//   - [Container.Contains]
func WithTag(tag any) TagOption {
	return tagOption{tag}
}

// TODO: Use dependency tag with Invoke

// WithDependencyTag is used to specify a tag for a dependency when calling [WithService].
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithService(NewUsersDB, di.WithTag("users")),
//		di.WithService(NewOrdersDB, di.WithTag("orders")),
//		di.WithService(NewUsersStore,
//			di.WithDependencyTag[*sql.DB]("users")
//		),
//		di.WithService(NewOrdersStore,
//			di.WithDependencyTag[*sql.DB]("orders")
//		),
//	)
func WithDependencyTag[T any](tag any) ServiceOption {
	depType := reflect.TypeFor[T]()

	return serviceOption(func(s service) error {
		funcSvc, ok := s.(*funcService)
		if !ok {
			// Option will be ignored
			return nil
		}

		for _, dep := range funcSvc.deps {
			if dep.Type == depType {
				dep.Tag = tag
				return nil
			}
		}
		return errors.Errorf("dependency %s not found", depType)
	})
}

type tagOption struct {
	tag any
}

func (o tagOption) applyService(s service) error {
	s.setTag(o.tag)
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
