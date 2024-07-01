package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// As registers the service as type Service when calling [WithService].
// This is useful when you want to register a service as an interface that it implements.
//
// This option will return an error if the service type is not assignable to type Service.
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithService(service.NewService,	// returns *service.Service
//			di.As[service.Interface](),	// register as interface
//			di.As[*service.Service](),	// also register as actual type
//		),
//		// ...
//	)
func As[Service any]() ServiceOption {
	return serviceOption(func(sr serviceRegistration) error {
		aliasType := reflect.TypeFor[Service]()

		err := sr.AddAlias(aliasType)
		if err != nil {
			return errors.Wrapf(err, "as %s", aliasType)
		}

		return nil
	})
}
