package di

import (
	"cmp"
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/sectrean/di-kit/internal/errors"
)

// ValidateContainer will validate that the given Container's services are all resolvable.
//
// This will verify that all service dependencies are registered and
// there are no dependency cycles.
// It will return an error with details if any issues are found.
//
// Scoped services are not validated on parent containers because dependencies
// may be registered with a child scope.
// Child scopes can be validated separately.
//
// This function is intended to be used in tests, not in production code.
func ValidateContainer(c *Container) error {
	v := newValidator(c)

	if err := v.Validate(); err != nil {
		return errors.Wrap(err, "di.ValidateContainer")
	}
	return nil
}

func newValidator(c *Container) *validator {
	return &validator{scope: c}
}

type validator struct {
	scope *Container
}

func (v *validator) Validate() error {
	var errs []error

	for svc := range v.services() {
		err := v.validateService(svc, newResolveVisitor())
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Sort errors so we have stable output despite iterating
	// on services from a map
	slices.SortFunc(errs, func(a, b error) int {
		return cmp.Compare(a.Error(), b.Error())
	})

	if err := errors.Join(errs...); err != nil {
		return err
	}

	return nil
}

func (v *validator) validateService(svc *service, visitor *resolveVisitor) error {
	deps := svc.Dependencies()
	if len(deps) == 0 {
		return nil
	}

	if !visitor.Enter(svc) {
		return newServiceValidationError(svc, errDependencyCycle)
	}
	defer visitor.Leave()

	var depErrs []error
	for depIndex, depKey := range deps {
		if depKey.Type == typeContext || depKey.Type == typeScope {
			continue
		}

		if isUnnamedSliceType(depKey.Type) {
			// A variadic parameter is optional
			optional := depIndex == len(deps)-1 && svc.Func().Type().IsVariadic()
			elemKey := serviceKey{
				Type: depKey.Type.Elem(),
				Tag:  depKey.Tag,
			}

			sliceSvcs := slices.Collect(v.scope.registeredServices(elemKey))
			if len(sliceSvcs) == 0 && !optional {
				depErrs = append(depErrs, newDependencyError(depKey, errServiceNotRegistered))
				continue
			}

			for _, depSvc := range sliceSvcs {
				if err := v.validateService(depSvc, visitor); err != nil {
					depErrs = append(depErrs, newDependencyError(depKey, err))
				}
			}
			continue
		}

		depSvc := v.scope.lookupService(depKey)
		if depSvc == nil {
			depErrs = append(depErrs, newDependencyError(depKey, errServiceNotRegistered))
			continue
		}

		if err := v.validateService(depSvc, visitor); err != nil {
			depErrs = append(depErrs, newDependencyError(depKey, err))
		}
	}

	if len(depErrs) > 0 {
		return newServiceValidationError(svc, depErrs...)
	}

	return nil
}

// services returns services to validate.
// A service is returned only once even if it was registered with multiple
// types or tags. Iteration order is not stable.
//
// Scoped services registered with ancestor containers are validated.
func (v *validator) services() iter.Seq[*service] {
	return func(yield func(*service) bool) {
		seen := make(map[*service]struct{})
		checkScoped := false // Don't check scoped services on this container--only ancestors

		for scope := v.scope; scope != nil; scope = scope.parent {
			for _, svcs := range scope.services {
				for _, svc := range svcs {
					if (svc.Lifetime() == Scoped) != checkScoped {
						continue
					}

					if _, ok := seen[svc]; ok {
						continue
					}
					seen[svc] = struct{}{}

					if !yield(svc) {
						return
					}
				}
			}

			checkScoped = true
		}
	}
}

func newServiceValidationError(svc *service, depErrs ...error) error {
	return &validationError{
		svc:     svc,
		depErrs: depErrs,
	}
}

type validationError struct {
	svc     *service
	depErrs []error
}

var _ error = (*validationError)(nil)

func (e *validationError) Error() string {
	var sb strings.Builder

	sb.WriteString("service ")
	sb.WriteString(e.svc.String())
	sb.WriteString(": ")

	for i, err := range e.depErrs {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Error())
	}

	return sb.String()
}

// func (e *validationError) Unwrap() []error { return e.depErrs }

func newDependencyError(key serviceKey, cause error) error {
	return &dependencyError{
		key:   key,
		cause: cause,
	}
}

type dependencyError struct {
	key   serviceKey
	cause error
}

var _ error = (*dependencyError)(nil)

func (e *dependencyError) Error() string {
	return fmt.Sprintf("dependency %s: %s", e.key, e.cause.Error())
}

// func (e *dependencyError) Unwrap() error { return e.cause }
