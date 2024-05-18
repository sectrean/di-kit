package di

import "reflect"

// As registers an alias for a service.
// Use with [RegisterFunc] or [RegisterValue].
func As[T any]() AliasOption {
	return serviceOption(func(s service) error {
		return s.AddAlias(reflect.TypeFor[T]())
	})
}

// AliasOption is an option that can be used when calling [RegisterFunc] or [RegisterValue].
type AliasOption interface {
	RegisterFuncOption
	RegisterValueOption
}
