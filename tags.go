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

// ServiceTagOption is used to specify the tag associated with a service when calling [WithService],
// [Resolve], [Container.Resolve], or [Container.Contains].
type ServiceTagOption interface {
	ServiceOption
	ResolveOption
	DecoratorOption
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
func WithTagged[Dependency any](tag any) DependencyOption {
	// Assign the tag to the first dependency of the right type that does not already have a tag.
	// If no dependency is found, an error is returned.
	//
	// We modify the slice items in place.
	return dependencyOption(func(deps []serviceKey) error {
		depType := reflect.TypeFor[Dependency]()

		for i := range deps {
			// Find the first dependency with the right type
			// Skip past any that have already been assigned a tag
			if deps[i].Type == depType && deps[i].Tag == nil {
				deps[i].Tag = tag
				return nil
			}
		}

		return errors.Errorf("WithTagged %s: parameter not found", depType)
	})
}

// DependencyOption is used to configure a service dependency when calling [WithService] or [Invoke].
type DependencyOption interface {
	ServiceOption
	InvokeOption
	DecoratorOption
}

type tagOption struct {
	tag any
}

func (o tagOption) applyServiceConfig(sc serviceConfig) error {
	sc.SetTag(o.tag)
	return nil
}

func (o tagOption) applyServiceKey(key serviceKey) serviceKey {
	return serviceKey{
		Type: key.Type,
		Tag:  o.tag,
	}
}

func (o tagOption) applyDecorator(d *decorator) error {
	return d.SetTag(o.tag)
}

var _ ServiceTagOption = tagOption{}

type dependencyOption func(deps []serviceKey) error

func (o dependencyOption) applyServiceConfig(sc serviceConfig) error {
	return o(sc.Dependencies())
}

func (o dependencyOption) applyDecorator(d *decorator) error {
	return o(d.deps)
}

func (o dependencyOption) applyInvokeConfig(c *invokeConfig) error {
	return o(c.deps)
}

var _ DependencyOption = dependencyOption(nil)
