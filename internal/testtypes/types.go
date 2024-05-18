package testtypes

import (
	"context"
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

type StructA struct{}

func (*StructA) A()                          {}
func (*StructA) Close(context.Context) error { return nil }

type StructB struct{}

func (*StructB) B()                    {}
func (*StructB) Close(context.Context) {}

func NewInterfaceA() InterfaceA {
	return &StructA{}
}

func NewStructAPtr() *StructA {
	return &StructA{}
}

func NewInterfaceB() InterfaceB {
	return &StructB{}
}

func NewStructBPtr() *StructB {
	return &StructB{}
}

func NewInterfaceADependsOnB(_ InterfaceB) InterfaceA {
	return &StructA{}
}

func NewInterfaceBDependsOnA(_ InterfaceA) InterfaceB {
	return &StructB{}
}
