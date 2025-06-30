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

func safeReflectValue(t reflect.Type, val any) reflect.Value {
	if val == nil {
		return reflect.Zero(t)
	}

	return reflect.ValueOf(val)
}

func isNil(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
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
