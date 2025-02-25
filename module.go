package di

import "slices"

// A Module is a collection of container options.
// It can be used to export a re-usable group of related services.
//
// Example:
//
//	var DependencyModule = di.Module{
//		di.WithService(NewDB),
//		di.WithService(NewStore),
//		di.WithService(NewService),
//	}
type Module []ContainerOption

func (Module) applyContainer(c *Container) error { return nil }
func (Module) order() optionOrder                { return 0 }

// WithModule applies the options in a module [Module] when calling [NewContainer] or [Container.NewScope].
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

func flattenModules(opts []ContainerOption) []ContainerOption {
	for i, opt := range opts {
		if mod, ok := opt.(Module); ok {
			opts = slices.Insert(opts, i+1, mod...)
		}
	}

	return opts
}
