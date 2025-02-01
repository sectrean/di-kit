package di

import (
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

// As registers the service as type Service when calling [WithService].
// This is useful when you want to register a service as an interface that it implements.
//
// This option will return an error if the service type is not assignable to type Service.
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithService(service.NewService,	// Function returns *service.Service
//			di.As[service.Interface](),	// Register as interface type
//			di.As[*service.Service](),	// Also register as pointer type
//		),
//		// ...
//	)
func As[Service any]() ServiceOption {
	return serviceOption(func(sc serviceConfig) error {
		aliasType := reflect.TypeFor[Service]()

		err := sc.AddAlias(aliasType)
		if err != nil {
			return errors.Wrapf(err, "As %s", aliasType)
		}

		return nil
	})
}
