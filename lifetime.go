package di

import "fmt"

// Lifetime specifies how services are created when resolved.
//
// Available lifetimes:
//   - [Singleton] specifies that a service is created once and subsequent requests return the same instance.
//   - [Transient] specifies that a service is created for each request.
//   - [Scoped] specifies that a service is created once per scope.
type Lifetime uint8

const (
	// Singleton specifies that a service is created once and subsequent requests to resolve return the same instance.
	//
	// This is the default lifetime for services.
	Singleton Lifetime = iota

	// Transient specifies that a service is created for each request.
	Transient Lifetime = iota

	// Scoped specifies that a service is created once per scope.
	Scoped Lifetime = iota
)

// WithLifetime is used to configure the lifetime of a service when calling [RegisterFunc].
//
// Example:
//
//	c, err := di.NewContainer(
//		di.RegisterFunc(NewService, di.WithLifetime(di.Transient)),
//		// Lifetime can also be used directly as an option
//		di.RegisterFunc(NewService, di.Transient),
//	)
func WithLifetime(lifetime Lifetime) LifetimeOption {
	return lifetime
}

// LifetimeOption is used to configure the lifetime of a service when calling [RegisterFunc].
type LifetimeOption interface {
	RegisterFuncOption
}

func (l Lifetime) applyFuncService(s *funcService) error {
	s.lifetime = l
	return nil
}

var _ LifetimeOption = Singleton

func (l Lifetime) String() string {
	switch l {
	case Singleton:
		return "Singleton"
	case Transient:
		return "Transient"
	case Scoped:
		return "Scoped"
	default:
		return fmt.Sprintf("Unknown Lifetime %d", l)
	}
}
