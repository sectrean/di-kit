package di

type ProvideOptions struct {
	// Lifetime controls the lifetime of an instance.
	// The default is Singleton.
	Lifetime Lifetime
}

type ProvideOption func(opts *ProvideOptions) error

func WithLifetime(l Lifetime) ProvideOption {
	return func(opts *ProvideOptions) error {
		opts.Lifetime = l
		return nil
	}
}
