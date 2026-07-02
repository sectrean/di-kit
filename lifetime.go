package di

import (
	"fmt"

	"github.com/sectrean/di-kit/internal/errors"
)

// Lifetime specifies how services are created when resolved.
//
// Use when registering a service with [WithService].
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

func (l Lifetime) applyService(s *service) error {
	switch l {
	case Singleton:
		// Singleton is always valid
	case Transient, Scoped:
		if s.IsValue() {
			// Value services can only be singletons because they are created outside of the container.
			return errors.Errorf("%s: invalid lifetime for value service", l.string())
		}

	default:
		return errors.Errorf("%s: invalid lifetime", l.string())
	}

	s.lifetime = l
	return nil
}

var _ ServiceOption = Singleton

func (l Lifetime) string() string {
	switch l {
	case Singleton:
		return "di.Singleton"
	case Transient:
		return "di.Transient"
	case Scoped:
		return "di.Scoped"
	default:
		return fmt.Sprintf("di.Lifetime(%d)", l)
	}
}
