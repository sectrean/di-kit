package di

import (
	"reflect"

	"github.com/google/uuid"
)

type Service interface {
	ID() uuid.UUID
	Type() reflect.Type
	GetValue(deps []any) (any, error)
	Lifetime() Lifetime
	Dependencies() []reflect.Type
	GetCloser(val any) (Closer, bool)
}
