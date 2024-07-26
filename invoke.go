package di

import (
	"context"
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// Invoke calls the given function with dependencies resolved from the provided Scope.
//
// The function may take any number of parameters. These dependencies must be registered with the Scope.
// The function may also accept a context.Context.
// The function may return an error. Any other return values are ignored.
func Invoke(ctx context.Context, s Scope, fn any, opts ...InvokeOption) error {
	fnType := reflect.TypeOf(fn)
	fnVal := reflect.ValueOf(fn)

	// Make sure fn is a function
	if fnType.Kind() != reflect.Func {
		return errors.Errorf("invoke %T: fn must be a function", fn)
	}

	config := &invokeConfig{
		fn: fnVal,
	}

	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		config.deps = append(config.deps, serviceKey{Type: paramType})
	}

	err := applyOptions(opts, func(opt InvokeOption) error {
		return opt.applyInvokeConfig(config)
	})
	if err != nil {
		return errors.Wrapf(err, "invoke %T", fn)
	}

	// Resolve deps from the Scope
	// Stop at the first error
	in := make([]reflect.Value, fnType.NumIn())
	for i, dep := range config.deps {
		var depVal any
		var depErr error

		switch {
		case dep.Type == typeContext:
			depVal = ctx
		case dep.Tag != nil:
			depVal, depErr = s.Resolve(ctx, dep.Type, WithTag(dep.Tag))
		default:
			depVal, depErr = s.Resolve(ctx, dep.Type)
		}

		if depErr != nil {
			return errors.Wrapf(depErr, "invoke %T", fn)
		}
		in[i] = safeVal(dep.Type, depVal)
	}

	// Check for a context error before we invoke the function
	if ctx.Err() != nil {
		return errors.Wrapf(ctx.Err(), "invoke %T", fn)
	}

	// Invoke the function
	out := fnVal.Call(in)

	// See if the function returns an error
	for i := 0; i < fnType.NumOut(); i++ {
		if fnType.Out(i) == typeError {
			err, _ := out[i].Interface().(error)
			return err
		}
	}

	return nil
}

// InvokeOption is used to configure the behavior of Invoke.
type InvokeOption interface {
	applyInvokeConfig(*invokeConfig) error
}

type invokeConfig struct {
	fn   reflect.Value
	deps []serviceKey
}
