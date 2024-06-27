package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// WithTag is used to specify the tag associated with a service.
//
// WithTag can be used with:
//   - [WithService]
//   - [Resolve]
//   - [MustResolve]
//   - [Container.Resolve]
//   - [Container.Contains]
func WithTag(tag any) ServiceTagOption {
	return tagOption{tag: tag}
}

// WithTagged is used to specify a tag for a service dependency when calling
// [WithService] or [Invoke].
//
// This option can be used multiple times to specify keys for function service dependencies.
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithService(db.NewPrimaryDB, di.WithTag(db.Primary)),
//		di.WithService(db.NewReplicaDB, di.WithTag(db.Replica)),
//		di.WithService(storage.NewReadWriteStore,
//			di.WithTagged[*db.DB](db.Primary),
//		),
//		di.WithService(storage.NewReadOnlyStore,
//			di.WithTagged[*db.DB](db.Replica),
//		),
//	)
//
// This option will return an error if the Service does not have a dependency of type Dependency.
func WithTagged[Dependency any](tag any) DependencyTagOption {
	return depTagOption{
		t:   reflect.TypeFor[Dependency](),
		tag: tag,
	}
}

// ServiceTagOption is used to specify the tag associated with a service when calling [WithService],
// [Resolve], [Container.Resolve], or [Container.Contains].
type ServiceTagOption interface {
	ServiceOption
	ResolveOption
}

// DependencyTagOption is used to specify a tag for a dependency when calling [WithService] or [Invoke].
type DependencyTagOption interface {
	ServiceOption
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

var _ ServiceTagOption = tagOption{}

type depTagOption struct {
	t   reflect.Type
	tag any
}

func (o depTagOption) applyDeps(deps []serviceKey) error {
	for i := 0; i < len(deps); i++ {
		// Find a dependency with the right type
		// Skip past any that have already been assigned a tag
		if deps[i].Type == o.t && deps[i].Tag == nil {
			deps[i].Tag = o.tag
			return nil
		}
	}
	return errors.Errorf("with tagged %s: argument not found", o.t)
}

func (o depTagOption) applyInvokeConfig(c *invokeConfig) error {
	return o.applyDeps(c.deps)
}

func (o depTagOption) applyService(s service) error {
	return o.applyDeps(s.Dependencies())
}

var _ DependencyTagOption = depTagOption{}
