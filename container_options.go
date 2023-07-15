package di

import (
	"reflect"

	"github.com/pkg/errors"
)

// TODO: Implement additional ContainerOptions
// - Validate dependencies--make sure all types are resolvable, no cycles
// - Use generated code: `di.UseCodegen()`

// ContainerOptions are used to create a Container.
type ContainerOptions struct {
	Services []Service
}

type ContainerOption func(opts *ContainerOptions) error

// Provide registers the given function or value with the container.
//
// The fnOrValue argument must be a function or a value.
// The function may take any number of arguments. These dependencies must be registered with the container.
// The function may also accept a context.Context.
// The function must return one or more services and optionally an error.
func Provide(fnOrValue any, opts ...ProvideOption) ContainerOption {
	return func(co *ContainerOptions) error {
		options := &ProvideOptions{
			Lifetime: Singleton,
		}
		for _, opt := range opts {
			err := opt(options)
			if err != nil {
				return errors.Wrapf(err, "registering %T", fnOrValue)
			}
		}

		svcs, err := newServices(fnOrValue, options)
		if err != nil {
			return errors.Wrapf(err, "registering %T", fnOrValue)
		}

		co.Services = append(co.Services, svcs...)

		return nil
	}
}

func newServices(fnOrValue any, options *ProvideOptions) ([]Service, error) {
	val := reflect.ValueOf(fnOrValue)
	var services []Service

	if val.Kind() == reflect.Func {
		s, err := newFuncService(val, options)
		if err != nil {
			return nil, errors.Wrapf(err, "registering func %v", val)
		}

		services = append(services, s)
	} else {
		s, err := newValueService(val, options)
		if err != nil {
			return nil, errors.Wrapf(err, "registering value %v", val)
		}

		services = append(services, s)
	}

	return services, nil
}
