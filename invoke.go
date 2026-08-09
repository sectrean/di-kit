package di

import (
	"context"
	"reflect"

	"github.com/sectrean/di-kit/internal/errors"
)

// Invoke calls the given function with parameters resolved from the given Scope.
//
// The function may take any number of parameters which will be resolved from the container,
// and may return any number of results.
// An [error] return parameter will be passed along and any other return parameters are ignored.
//
// A variadic parameter is treated as an optional slice dependency: if no services are
// registered for the element type, the function is called with an empty variadic argument.
func Invoke(ctx context.Context, s Scope, fn any, opts ...InvokeOption) error {
	config, err := newInvokeConfig(fn, opts...)
	if err != nil {
		return errors.Wrap(err, "di.Invoke")
	}

	// Resolve deps from the Scope
	deps := config.Dependencies()
	in := make([]reflect.Value, len(deps))

	for i, dep := range deps {
		var depVal any
		var depErr error

		var opts []ResolveOption
		if dep.Tag != nil {
			opts = append(opts, WithTag(dep.Tag))
		}

		switch {
		case dep.Type == typeContext:
			depVal = ctx

		case dep.Type == typeScope:
			depVal = s // no need to wrap this scope; we're not within a resolve operation

		// If the function accepts variadic args, the final argument is optional.
		case config.Type().IsVariadic() && i == len(deps)-1 && !s.Contains(dep.Type, opts...):
			// Create an empty slice
			depVal = reflect.MakeSlice(dep.Type, 0, 0).Interface()

		default:
			depVal, depErr = s.Resolve(ctx, dep.Type, opts...)
		}

		if depErr != nil {
			// Stop at the first error
			return errors.Wrap(depErr, "di.Invoke")
		}
		in[i] = safeReflectValue(dep.Type, depVal)
	}

	// Check for a context error before we invoke the function
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "di.Invoke")
	}

	// Invoke the function
	var out []reflect.Value
	if config.Type().IsVariadic() {
		out = config.Func().CallSlice(in)
	} else {
		out = config.Func().Call(in)
	}

	// Return the first error return value, if any.
	// Don't wrap the error, return it as-is.
	for i := range config.Type().NumOut() {
		if config.Type().Out(i) == typeError {
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

func newInvokeConfig(fn any, opts ...InvokeOption) (*invokeConfig, error) {
	fnVal := reflect.ValueOf(fn)
	if isNil(fnVal) {
		return nil, errors.New("fn is nil")
	}

	// Make sure fn is a function
	fnType := fnVal.Type()
	if fnType.Kind() != reflect.Func {
		return nil, errors.New("fn must be a function")
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
	if err := config.apply(opts...); err != nil {
		return nil, err
	}
	return config, nil
}

type invokeConfig struct {
	fn   reflect.Value
	deps []serviceKey
}

func (c *invokeConfig) apply(opts ...InvokeOption) error {
	return applyOptions(opts, func(o InvokeOption) error { return o.applyInvokeConfig(c) })
}

func (c invokeConfig) Type() reflect.Type         { return c.fn.Type() }
func (c invokeConfig) Func() reflect.Value        { return c.fn }
func (c invokeConfig) Dependencies() []serviceKey { return c.deps }
