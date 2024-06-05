package di

// ContainerOption is used to configure a new [Container] when calling [NewContainer]
// or [Container.NewScope].
type ContainerOption interface {
	applyContainer(*Container) error
}

type containerOption func(*Container) error

func (o containerOption) applyContainer(c *Container) error {
	return o(c)
}
