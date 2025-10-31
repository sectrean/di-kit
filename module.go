package di

// A Module is a collection of container options.
// It can be used to create a re-usable collection of related services.
//
// Example:
//
//	var DependencyModule = di.Module{
//		di.WithService(NewDB),
//		di.WithService(NewStore),
//		di.WithService(NewService),
//	}
type Module []ContainerOption

func (m Module) applyContainer(c *Container) error {
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
//		di.WithModule(DependencyModule), // var DependencyModule di.Module
//		di.WithService(NewHandler), // NewHandler(*slog.Logger, *db.DB) *Handler
//	)
func WithModule(m Module) ContainerOption {
	return m
}
