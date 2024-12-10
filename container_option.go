package di

import (
	"github.com/johnrutherford/di-kit/internal/errors"
)

// ContainerOption is used to configure a new [Container] when calling [NewContainer]
// or [Container.NewScope].
type ContainerOption interface {
	order() optionOrder
	applyContainer(*Container) error
}

// WithOptions applies multiple options when calling [NewContainer] or [Container.NewScope].
// This can be useful if you want to create re-usable collections of services.
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithOptions(app.CommonServices()), // CommonServices() []di.ContainerOption
//		di.WithService(NewHandler), // NewHandler(*slog.Logger, *db.DB) *Handler
//	)
func WithOptions(opts []ContainerOption) ContainerOption {
	return newContainerOption(orderOptions, func(c *Container) error {
		var errs []error
		for _, opt := range opts {
			if err := opt.applyContainer(c); err != nil {
				errs = append(errs, err)
			}
		}

		if err := errors.Join(errs...); err != nil {
			return errors.Wrap(err, "with options")
		}
		return nil
	})
}

type optionOrder int8

const (
	orderOptions   optionOrder = 5
	orderService   optionOrder = 10
	orderDecorator optionOrder = 20
)

func newContainerOption(order optionOrder, fn func(*Container) error) ContainerOption {
	return containerOption{fn: fn, ord: order}
}

type containerOption struct {
	fn  func(*Container) error
	ord optionOrder
}

func (o containerOption) order() optionOrder {
	return o.ord
}

func (o containerOption) applyContainer(c *Container) error {
	return o.fn(c)
}
