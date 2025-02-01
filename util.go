package di

import (
	"cmp"
	"context"
	"reflect"
	"slices"

	"github.com/sectrean/di-kit/internal/errors"
)

// These are commonly used types.
var (
	typeError   = reflect.TypeFor[error]()
	typeContext = reflect.TypeFor[context.Context]()
	typeScope   = reflect.TypeFor[Scope]()
)

func safeVal(t reflect.Type, val any) reflect.Value {
	if val == nil {
		return reflect.Zero(t)
	}

	return reflect.ValueOf(val)
}

// Apply functional options and join any errors together.
func applyOptions[O any](opts []O, f func(O) error) error {
	var errs []error

	for _, o := range opts {
		err := f(o)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func applyContainerOptions(c *Container, opts []ContainerOption) error {
	// Flatten any modules before sorting and applying options
	for i := 0; i < len(opts); i++ {
		if mod, ok := opts[i].(Module); ok {
			opts = append(opts, mod...)
		}
	}

	// Sort options by precedence
	// Use stable sort because the registration order of services matters
	slices.SortStableFunc(opts, func(a, b ContainerOption) int {
		return cmp.Compare(a.order(), b.order())
	})

	var errs []error
	for _, o := range opts {
		err := o.applyContainer(c)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
