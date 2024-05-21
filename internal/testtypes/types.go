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

type StructC struct{}

func (*StructC) C()           {}
func (*StructC) Close() error { return nil }

type StructD struct{}

func (*StructD) D()     {}
func (*StructD) Close() {}

func NewInterfaceA() InterfaceA {
	return &StructA{}
}

func NewStructAPtr() *StructA {
	return &StructA{}
}

func NewInterfaceB(InterfaceA) InterfaceB {
	return &StructB{}
}

func NewStructBPtr(*StructA) *StructB {
	return &StructB{}
}

func NewInterfaceC(InterfaceA, InterfaceB) InterfaceC {
	return &StructC{}
}

func NewStructCPtr(*StructA, *StructB) *StructC {
	return &StructC{}
}

func NewInterfaceD(InterfaceA, InterfaceB, InterfaceC) InterfaceD {
	return &StructD{}
}

func NewStructDPtr(*StructA, *StructB, *StructC) *StructD {
	return &StructD{}
}
