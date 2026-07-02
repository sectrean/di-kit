package di

import (
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

// WithTag is used to specify a tag associated with a service.
// Services are registered with a `nil` tag by default.
//
// When registering a service, this option can be used multiple times to
// associate multiple tags with a service.
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

// ServiceTagOption is used to specify the tag associated with a service when calling [WithService],
// [Resolve], [Container.Resolve], or [Container.Contains].
type ServiceTagOption interface {
	ServiceOption
	ResolveOption
}

// WithTagged is used to specify a tag for a service dependency when calling
// [WithService] or [Invoke].
//
// This option will return an error if the service does not have a dependency of type *Dependency*.
//
// This option can be used multiple times to specify keys for function service dependencies.
// Multiple dependencies of the same type are allowed, but they should each have a non-nil tag specified.
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
