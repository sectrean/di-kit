package di

import (
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

// WithTag is used to specify a tag associated with a service.
//
// When registering a service, WithTag can be used multiple times to associate multiple tags with a service.
// See also [WithDefaultTag].
//
// WithTag can be used with:
//   - [WithService]
//   - [Resolve]
//   - [MustResolve]
//   - [Contains]
//   - [Container.Resolve]
//   - [Container.Contains]
func WithTag(tag any) ServiceTagOption {
	return tagOption{Tag: tag}
}

// WithDefaultTag is used to associate the default tag with a service.
//
// This is useful when you register a service with a tag, but you also want the service to
// be resolved when no tag is specified.
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithService(client.NewReadClient,
//			di.WithTag(client.Read),
//		),
//		di.WithService(client.NewWriteClient,
//			di.WithTag(client.Write),
//			di.WithDefaultTag(),
//		),
//		...
//	)
func WithDefaultTag() ServiceOption {
	return WithTag(nil)
}

// ServiceTagOption is used to specify the tag associated with a service when calling [WithService],
// [Resolve], [Container.Resolve], or [Container.Contains].
type ServiceTagOption interface {
	ServiceOption
	ResolveOption
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
// This option will return an error if the service does not have a dependency of type *Dependency*.
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
}

type tagOption struct {
	Tag any
}

func (o tagOption) applyService(s *service) error {
	s.tags = append(s.tags, o.Tag)
	return nil
}

func (o tagOption) applyServiceKey(key serviceKey) serviceKey {
	return serviceKey{
		Type: key.Type,
		Tag:  o.Tag,
	}
}

var _ ServiceTagOption = tagOption{}

type dependencyOption func(deps []serviceKey) error

func (o dependencyOption) applyService(s *service) error {
	return o(s.Dependencies())
}

func (o dependencyOption) applyInvokeConfig(c *invokeConfig) error {
	return o(c.deps)
}

var _ DependencyOption = dependencyOption(nil)
