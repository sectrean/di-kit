package di

// A Module is a collection of container options.
// It can be used to export a re-usable group of related services.
//
// Example:
//
//	func DependencyModule() di.Module {
//		return di.Module{
//	        di.WithService(NewDB),
//	        di.WithService(NewStore),
//	        di.WithService(NewService),
//	    }
//	}
type Module []ContainerOption

func (Module) applyContainer(c *Container) error { return nil }
func (Module) order() optionOrder                { return 0 }

// WithModule applies a [Module] when calling [NewContainer] or [Container.NewScope].
// This can be useful if you want to create re-usable collections of services.
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithModule(app.DependencyModule()), // DependencyModule() di.Module
//		di.WithService(NewHandler), // NewHandler(*slog.Logger, *db.DB) *Handler
//	)
func WithModule(m Module) ContainerOption {
	return m
}
