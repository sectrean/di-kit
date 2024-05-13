package di

import (
	"context"
	"reflect"
)

// TypeOf returns the [reflect.Type] of T.
func TypeOf[T any]() reflect.Type {
	var t T
	return reflect.TypeOf(&t).Elem()
}

// Common types used in the package
var (
	errorType   = TypeOf[error]()
	contextType = TypeOf[context.Context]()
	scopeType   = TypeOf[Scope]()
)
