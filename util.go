package di

import (
	"context"
	"reflect"

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
