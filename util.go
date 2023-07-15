package di

import (
	"reflect"
)

func TypeOf[T any]() reflect.Type {
	var t T
	return reflect.TypeOf(t)
}

func Map[T, U any](in []T, fn func(T) U) []U {
	if in == nil {
		return nil
	}

	out := make([]U, len(in))
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}
