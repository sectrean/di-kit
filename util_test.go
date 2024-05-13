package di

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func Test_TypeOf(t *testing.T) {
	tests := []struct {
		name string
		call func() reflect.Type
		want string
	}{
		{
			name: "error",
			call: TypeOf[error],
			want: "error",
		},
		{
			name: "Context",
			call: TypeOf[context.Context],
			want: "context.Context",
		},
		{
			name: "Scope interface",
			call: TypeOf[Scope],
			want: "di.Scope",
		},
		{
			name: "public struct",
			call: TypeOf[Container],
			want: "di.Container",
		},
		{
			name: "public struct pointer",
			call: TypeOf[*Container],
			want: "*di.Container",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.call()
			assert.Equal(t, tt.want, got.String())
		})
	}
}
