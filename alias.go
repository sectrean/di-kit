package di

// As registers an alias for a service.
func As[T any]() AliasOption {
	return serviceOption(func(s service) error {
		aliasType := TypeOf[T]()
		return s.AddAlias(aliasType)
	})
}

// AliasOption is an interface that can be used to register an alias for a service.
type AliasOption interface {
	RegisterFuncOption
	RegisterValueOption
}
