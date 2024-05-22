package di

import "fmt"

// Lifetime specifies how services are created when resolved.
//
// Use when registering a service with [Register].
//
// Available lifetimes:
//   - [Singleton] specifies that a service is created once and subsequent requests return the same instance.
//   - [Transient] specifies that a service is created for each request.
//   - [Scoped] specifies that a service is created once per scope.
//
// Example:
//
//	c, err := di.NewContainer(
//		di.Register(NewService, di.Transient),
//		di.Register(NewRequestService, di.Scoped),
//	)
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

func (l Lifetime) applyService(s service) error {
	s.setLifetime(l)
	return nil
}

var _ RegisterOption = Singleton

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
