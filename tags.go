package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// WithTag is used to specify the tag associated with a service.
//
// WithTag can be used with:
//   - [Register]
//   - [Resolve]
//   - [MustResolve]
//   - [Container.Resolve]
//   - [Container.Contains]
func WithTag(tag any) TagOption {
	return tagOption{tag}
}

// WithDependencyTag is used to specify a tag for a dependency when calling
// [Register] or [Invoke].
//
// Example:
//
//	c, err := di.NewContainer(
//		di.Register(NewUsersDB, di.WithTag("users")),
//		di.Register(NewOrdersDB, di.WithTag("orders")),
//		di.Register(NewUsersStore,
//			di.WithDependencyTag[*sql.DB]("users")
//		),
//		di.Register(NewOrdersStore,
//			di.WithDependencyTag[*sql.DB]("orders")
//		),
//	)
func WithDependencyTag[T any](tag any) DependencyTagOption {
	return depTagOption{
		t:   reflect.TypeFor[T](),
		tag: tag,
	}
}

// TagOption is used to specify the tag associated with a service when calling [Register],
// [Resolve], [Container.Resolve], or [Container.Contains].
type TagOption interface {
	RegisterOption
	ServiceOption
}

// DependencyTagOption is used to specify a tag for a dependency when calling [Register] or [Invoke].
type DependencyTagOption interface {
	RegisterOption
	InvokeOption
}

type tagOption struct {
	tag any
}

func (o tagOption) applyService(s service) error {
	s.setTag(o.tag)
	return nil
}

func (o tagOption) applyServiceKey(key serviceKey) serviceKey {
	return serviceKey{
		Type: key.Type,
		Tag:  o.tag,
	}
}

var _ TagOption = tagOption{}

type depTagOption struct {
	t   reflect.Type
	tag any
}

func (o depTagOption) applyDeps(deps []serviceKey) error {
	for i := 0; i < len(deps); i++ {
		if deps[i].Type == o.t {
			deps[i].Tag = o.tag
			return nil
		}
	}
	return errors.Errorf("dependency %s not found", o.t)
}

func (o depTagOption) applyInvokeConfig(c *invokeConfig) error {
	return o.applyDeps(c.deps)
}

func (o depTagOption) applyService(s service) error {
	fs, ok := s.(*funcService)
	if !ok {
		// Option will be ignored
		return nil
	}

	return o.applyDeps(fs.deps)
}

var _ DependencyTagOption = depTagOption{}
