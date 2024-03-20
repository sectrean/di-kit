package di

import (
	"reflect"
)

// TypeOf returns the [reflect.Type] of T.
func TypeOf[T any]() reflect.Type {
	var t T
	return reflect.TypeOf(&t).Elem()
}

// Must panics if err is not nil.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
