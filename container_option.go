package di

import (
	"github.com/johnrutherford/di-kit/internal/errors"
)

// ContainerOption is used to configure a new Container when calling [NewContainer].
type ContainerOption interface {
	applyContainer(*Container) error
}

type containerOption func(*Container) error

func (o containerOption) applyContainer(c *Container) error {
	return o(c)
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
//
// This option should come before any other options.
func WithParent(parent *Container) ContainerOption {
	return containerOption(func(c *Container) error {
		lock := parent.closeMu.RLock()
		defer parent.closeMu.RUnlock(lock)

		if parent.closed {
			return errors.Wrap(ErrContainerClosed, "with parent")
		}

		if c.parent != nil {
			return errors.New("with parent: parent already set")
		}

		c.parent = parent

		if c.services == nil {
			c.services = parent.services
		} else {
			// Copy services from parent
			for k, v := range parent.services {
				if _, ok := c.services[k]; !ok {
					c.services[k] = v
				}
			}
		}

		return nil
	})
}
