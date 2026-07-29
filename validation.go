package di

import (
	"fmt"
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
	var errs []error
	svcProblems := make(map[*service]string)

	for _, svcs := range c.services {
		for _, svc := range svcs {
			if svc.Lifetime() == Scoped {
				// Scoped services are not validated
				continue
			}

			prob := validateService(c, svc, svcProblems, newResolveVisitor())
			if prob != "" {
				errs = append(errs, errors.Errorf("service %s: %s", svc, prob))
			}
		}
	}

	if c.parent != nil {
		// Validate scoped services on the parent Container
		for _, svcs := range c.parent.services {
			for _, svc := range svcs {
				if svc.Lifetime() != Scoped {
					// Now we only want the scoped services
					continue
				}

				prob := validateService(c, svc, svcProblems, newResolveVisitor())
				if prob != "" {
					errs = append(errs, errors.Errorf("service %s: %s", svc, prob))
				}
			}
		}
	}

	if err := errors.Join(errs...); err != nil {
		return errors.Wrap(err, "di.ValidateContainer")
	}

	return nil
}

func validateService(c *Container, svc *service, svcProblems map[*service]string, visitor *resolveVisitor) string {
	if prob, ok := svcProblems[svc]; ok {
		return prob
	}

	deps := svc.Dependencies()
	if len(deps) == 0 {
		svcProblems[svc] = ""
		return ""
	}

	if !visitor.Enter(svc) {
		return errDependencyCycle.Error()
	}
	defer visitor.Leave()

	var problems []string
	for _, depKey := range deps {
		if depKey.Type == typeContext || depKey.Type == typeScope {
			continue
		}

		if isUnnamedSliceType(depKey.Type) {
			if svc.Func().Type().IsVariadic() {
				// If the service is variadic, registration is optional
				continue
			}

			// Check that the element type is registered
			depKey.Type = depKey.Type.Elem()
		}

		depSvc := c.lookupService(depKey)
		if depSvc == nil {
			prob := fmt.Sprintf("dependency %s: service not registered", depKey)
			problems = append(problems, prob)
			continue
		}

		prob := validateService(c, depSvc, svcProblems, visitor)
		if prob != "" {
			problems = append(problems, fmt.Sprintf("dependency %s: %s", depKey, prob))
		}
	}

	if len(problems) > 0 {
		probs := strings.Join(problems, "; ")
		svcProblems[svc] = probs
		return probs
	}

	return ""
}
