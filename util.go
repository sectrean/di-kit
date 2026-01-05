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

func isNil(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:
		return v.IsNil()
	}

	return false
}

// applyOptions applies functional options and joins any errors together.
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

func isUnnamedSliceType(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.PkgPath() == "" && t.Name() == ""
}
