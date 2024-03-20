package di

// TODO: Implement additional ContainerOptions
// - Validate dependencies--make sure all types are resolvable, no cycles
// - Use generated code: `di.UseCodegen()`

// ContainerOptions are used to create a Container.
type ContainerOptions struct {
	parent   *Container
	services []Service
}

// ContainerOption is used to configure a Container.
type ContainerOption func(*ContainerOptions) error

// WithParent can be used to set the parent scope of the container.
func WithParent(parent *Container) ContainerOption {
	return func(co *ContainerOptions) error {
		co.parent = parent
		return nil
	}
}

// WithService registers the given function or value with the container.
//
// The fnOrValue argument must be a function or a value.
// The function may take any number of arguments. These dependencies must be registered with the container.
// The function may also accept a [context.Context].
// The function must return a service and optionally an error.
func WithService(fnOrValue any, opts ...ServiceOption) ContainerOption {
	return func(co *ContainerOptions) error {
		svc, err := NewService(fnOrValue, opts...)
		if err != nil {
			return err
		}

		co.services = append(co.services, svc)

		return nil
	}
}
