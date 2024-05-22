package di

import (
	"github.com/johnrutherford/di-kit/internal/errors"
)

// ContainerOption is used to configure a new Container.
type ContainerOption interface {
	applyContainer(*Container) error
}

// TODO: Implement additional Container options:
// - Validate dependencies--make sure all types are resolvable, no cycles

type containerOption func(*Container) error

func (f containerOption) applyContainer(c *Container) error {
	return f(c)
}

// WithParent can be used to create a new Container with a child scope.
//
// The child Container will inherit all registered services from the parent Container.
// The child Container will use a new scope for resolving [Scoped] services.
//
// Example:
//
//	childScope, err := NewContainer(
//		WithParent(c),
//		Register(valueForChildContainer),
//	)
func WithParent(parent *Container) ContainerOption {
	return containerOption(func(c *Container) error {
		if parent.closed.Load() {
			return errors.Wrap(ErrContainerClosed, "with parent")
		}

		c.parent = parent
		return nil
	})
}
