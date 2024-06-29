package di

// ContainerOption is used to configure a new [Container] when calling [NewContainer]
// or [Container.NewScope].
type ContainerOption interface {
	order() optionOrder
	applyContainer(*Container) error
}

type optionOrder int8

const (
	orderService   optionOrder = 10
	orderDecorator optionOrder = 20
)

func newContainerOption(order optionOrder, fn func(*Container) error) ContainerOption {
	return containerOption{order, fn}
}

type containerOption struct {
	ord optionOrder
	fn  func(*Container) error
}

func (o containerOption) order() optionOrder {
	return o.ord
}

func (o containerOption) applyContainer(c *Container) error {
	return o.fn(c)
}
