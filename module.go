package di

// A Module is a collection of container options.
// It can be used to create a re-usable collection of related services.
//
// Example:
//
//	var Deps = di.Module{
//		di.WithService(NewLogger),
//		di.WithService(NewDB),
//	}
type Module []ContainerOption

func (m Module) applyContainer(c *Container) error {
	// Apply each option contained in this module
	return applyOptions(m, func(o ContainerOption) error {
		return o.applyContainer(c)
	})
}

var _ ContainerOption = Module{}

// WithModule applies the container options in a Module when calling [NewContainer] or [Container.NewScope].
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithModule(common.Deps),
//		di.WithModule(service.Deps),
//	)
//
//	// Can also be used directly as an option
//	c, err := di.NewContainer(common.Deps, service.Deps)
func WithModule(m Module) ContainerOption {
	return m
}
