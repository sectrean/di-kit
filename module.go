package di

import "github.com/sectrean/di-kit/internal/errors"

// A Module is a collection of container options.
// It can be used to create a re-usable collection of related services.
//
// Example:
//
//	var Deps = di.Module{
//		common.Deps,
//		di.WithService(storage.NewStorage),
//		di.WithService(service.NewService),
//	}
//
//	func main() {
//		c, err := di.NewContainer(Deps)
//		//...
//	}
type Module []ContainerOption

func (m Module) applyContainer(c *Container) error {
	// Apply each option contained in this module
	err := applyOptions(m, func(o ContainerOption) error {
		return o.applyContainer(c)
	})
	if err != nil {
		return errors.Wrap(err, "di.Module")
	}

	return nil
}

var _ ContainerOption = Module{}
