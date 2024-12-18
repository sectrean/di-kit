package di

import "fmt"

// Lifetime specifies how services are created when resolved.
//
// Use when registering a service with [WithService].
//
// Available lifetimes:
//   - [SingletonLifetime] specifies that a service is created once and subsequent requests return the same instance.
//   - [TransientLifetime] specifies that a service is created for each request.
//   - [ScopedLifetime] specifies that a service is created once per scope.
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithService(NewService, di.Transient),
//		di.WithService(NewRequestService, di.Scoped),
//	)
type Lifetime uint8

const (
	// SingletonLifetime specifies that a service is created once and subsequent requests to resolve return the same instance.
	//
	// This is the default lifetime for services.
	SingletonLifetime Lifetime = iota

	// TransientLifetime specifies that a service is created for each request.
	TransientLifetime Lifetime = iota

	// ScopedLifetime specifies that a service is created once per scope.
	ScopedLifetime Lifetime = iota
)

func (l Lifetime) applyServiceConfig(sc serviceConfig) error {
	sc.SetLifetime(l)
	return nil
}

var _ ServiceOption = SingletonLifetime

func (l Lifetime) String() string {
	switch l {
	case SingletonLifetime:
		return "Singleton"
	case TransientLifetime:
		return "Transient"
	case ScopedLifetime:
		return "Scoped"
	default:
		return fmt.Sprintf("Unknown Lifetime %d", l)
	}
}
