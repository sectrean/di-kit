package testtypes

import (
	"context"
	"net/http"
	"reflect"
)

var (
	TypeStructA    = reflect.TypeFor[StructA]()
	TypeStructAPtr = reflect.TypeFor[*StructA]()
	TypeInterfaceA = reflect.TypeFor[InterfaceA]()

	TypeStructB    = reflect.TypeFor[StructB]()
	TypeStructBPtr = reflect.TypeFor[*StructB]()
	TypeInterfaceB = reflect.TypeFor[InterfaceB]()

	TypeStructC    = reflect.TypeFor[StructC]()
	TypeStructCPtr = reflect.TypeFor[*StructC]()
	TypeInterfaceC = reflect.TypeFor[InterfaceC]()

	TypeStructD    = reflect.TypeFor[StructD]()
	TypeStructDPtr = reflect.TypeFor[*StructD]()
	TypeInterfaceD = reflect.TypeFor[InterfaceD]()
)

type InterfaceA interface {
	A()
	Close(context.Context) error
}

type InterfaceB interface {
	B()
	Close(context.Context)
}

type InterfaceC interface {
	C()
	Close() error
}

type InterfaceD interface {
	D()
	Close()
}

type StructA struct {
	Tag any
}

func (StructA) A()                          {}
func (StructA) Close(context.Context) error { return nil }

type StructB struct{}

func (StructB) B()                    {}
func (StructB) Close(context.Context) {}

type StructC struct{}

func (StructC) C()           {}
func (StructC) Close() error { return nil }

type StructD struct{}

func (StructD) D()     {}
func (StructD) Close() {}

type CustomError struct{}

func (e *CustomError) Error() string {
	return "custom error"
}

type HTTPMiddleware func(http.Handler) http.Handler

func NewMiddleware() HTTPMiddleware {
	return func(next http.Handler) http.Handler { return next }
}

type CustomMap map[string]any

type CustomStringCollection []string

type CustomString string

func NewInterfaceA() InterfaceA {
	return &StructA{}
}

func NewInterfaceAStruct() InterfaceA {
	return StructA{}
}

func NewStructAPtr() *StructA {
	return &StructA{}
}

func NewInterfaceB(InterfaceA) InterfaceB {
	return &StructB{}
}

func NewInterfaceBStruct(InterfaceA) InterfaceB {
	return StructB{}
}

func NewStructBPtr(*StructA) *StructB {
	return &StructB{}
}

func NewInterfaceC(InterfaceA, InterfaceB) InterfaceC {
	return &StructC{}
}

func NewInterfaceCStruct(InterfaceA, InterfaceB) InterfaceC {
	return StructC{}
}

func NewStructCPtr(*StructA, *StructB) *StructC {
	return &StructC{}
}

func NewInterfaceD(InterfaceA, InterfaceB, InterfaceC) InterfaceD {
	return &StructD{}
}

func NewInterfaceDStruct(InterfaceA, InterfaceB, InterfaceC) InterfaceD {
	return StructD{}
}

func NewStructDPtr(*StructA, *StructB, *StructC) *StructD {
	return &StructD{}
}
