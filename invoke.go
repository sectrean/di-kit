package di

import (
	"context"
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

// Invoke calls the given function with parameters resolved from the provided Scope.
//
// The function may take any number of parameters which will be resolved from the container,
// and may return any number of results.
// An [error] return parameter will be passed along and any other return parameters are ignored.
func Invoke(ctx context.Context, s Scope, fn any, opts ...InvokeOption) error {
	fnType := reflect.TypeOf(fn)
	fnVal := reflect.ValueOf(fn)

	// Make sure fn is a function
	if fnType.Kind() != reflect.Func {
		return errors.Errorf("di.Invoke %T: fn must be a function", fn)
	}

	// Get the dependencies
	deps := make([]serviceKey, fnType.NumIn())
	for i := range fnType.NumIn() {
		deps[i] = serviceKey{
			Type: fnType.In(i),
		}
	}

	// Create a config struct so we can apply options
	config := &invokeConfig{
		fn:   fnVal,
		deps: deps,
	}

	// Apply options to the config
	err := applyOptions(opts, func(opt InvokeOption) error {
		return opt.applyInvokeConfig(config)
	})
	if err != nil {
		return errors.Wrapf(err, "di.Invoke %T", fn)
	}

	// Resolve deps from the Scope
	in := make([]reflect.Value, fnType.NumIn())
	for i, dep := range config.deps {
		var depVal any
		var depErr error

		switch {
		case dep.Type == typeContext:
			depVal = ctx
		case dep.Type == typeScope:
			depVal = s
		case dep.Tag != nil:
			depVal, depErr = s.Resolve(ctx, dep.Type, WithTag(dep.Tag))
		default:
			depVal, depErr = s.Resolve(ctx, dep.Type)
		}

		if depErr != nil {
			// Stop at the first error
			return errors.Wrapf(depErr, "di.Invoke %T", fn)
		}
		in[i] = safeVal(dep.Type, depVal)
	}

	// Check for a context error before we invoke the function
	if ctx.Err() != nil {
		return errors.Wrapf(ctx.Err(), "di.Invoke %T", fn)
	}

	// Invoke the function
	out := fnVal.Call(in)

	// Return the first error return value, if any.
	// Don't wrap the error, return it as-is.
	for i := range fnType.NumOut() {
		if fnType.Out(i) == typeError {
			err, _ := out[i].Interface().(error)
			return err
		}
	}

	return nil
}

// InvokeOption is used to configure the behavior of Invoke.
//
// Available options:
//   - WithTagged
type InvokeOption interface {
	applyInvokeConfig(*invokeConfig) error
}

type invokeConfig struct {
	fn   reflect.Value
	deps []serviceKey
}
