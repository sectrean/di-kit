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
	return withTagOption{Tag: tag}
}

// ServiceTagOption is used to specify the tag associated with a service when calling [WithService],
// [Resolve], [Container.Resolve], or [Container.Contains].
type ServiceTagOption interface {
	ServiceOption
	ResolveOption
}

// WithTagged is used to specify a tag to use when resolving a dependency.
// This can be used when calling [WithService] or [Invoke].
//
// This option will return an error if the function does not have a parameter of type Dependency.
//
// This option can be used multiple times to specify keys for different dependencies.
// Multiple dependencies of the same type are allowed, but they should each have a non-nil tag specified.
func WithTagged[Dependency any](tag any) DependencyTagOption {
	return withTaggedOption{
		Type: reflect.TypeFor[Dependency](),
		Tag:  tag,
	}
}

// DependencyTagOption is used to configure a service dependency when calling [WithService] or [Invoke].
type DependencyTagOption interface {
	ServiceOption
	InvokeOption
}

func validateTagType(t reflect.Type) error {
	if t == nil {
		return nil
	}

	if !t.Comparable() {
		return errors.Errorf("invalid tag type %s: type must be comparable", t)
	}

	return nil
}

type withTagOption struct {
	Tag any
}

func (o withTagOption) validate() error {
	t := reflect.TypeOf(o.Tag)

	err := validateTagType(t)
	if err != nil {
		return errors.Wrap(err, "di.WithTag")
	}

	return nil
}

func (o withTagOption) applyService(s *service) error {
	if err := o.validate(); err != nil {
		return err
	}

	s.tags = append(s.tags, o.Tag)
	return nil
}

func (o withTagOption) applyServiceKey(key serviceKey) (serviceKey, error) {
	if err := o.validate(); err != nil {
		return key, err
	}

	return serviceKey{
		Type: key.Type,
		Tag:  o.Tag,
	}, nil
}

var _ ServiceTagOption = withTagOption{}

type withTaggedOption struct {
	Type reflect.Type
	Tag  any
}

func (o withTaggedOption) validate() error {
	tagType := reflect.TypeOf(o.Tag)

	err := validateTagType(tagType)
	if err != nil {
		return errors.Wrapf(err, "di.WithTagged[%s]", o.Type)
	}

	return nil
}

// apply assigns the tag to the first dependency of the right type that does not already have a tag.
// If no dependency is found, an error is returned.
//
// The slice items are modified in place.
func (o withTaggedOption) apply(deps []serviceKey) error {
	if err := o.validate(); err != nil {
		return err
	}

	for i := range deps {
		// Find the first dependency with the right type
		// Skip past any that have already been assigned a tag
		if deps[i].Type == o.Type && deps[i].Tag == nil {
			deps[i].Tag = o.Tag
			return nil
		}
	}

	return errors.Errorf("di.WithTagged[%s]: parameter not found", o.Type)
}

func (o withTaggedOption) applyService(s *service) error {
	return o.apply(s.Dependencies())
}

func (o withTaggedOption) applyInvokeConfig(c *invokeConfig) error {
	return o.apply(c.deps)
}

var _ DependencyTagOption = withTaggedOption{}
