package testtypes

type InterfaceA interface {
	A()
}

type InterfaceB interface {
	B()
}

type StructA struct{}

func (*StructA) A() {}

type StructB struct{}

func (*StructB) B() {}

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
