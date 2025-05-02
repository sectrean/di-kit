package testtypes

type Factory struct {
	count int
}

func (f *Factory) NewStructA() *StructA {
	a := &StructA{
		Tag: f.count,
	}
	f.count++

	return a
}

func (f *Factory) NewInterfaceA() InterfaceA {
	return f.NewStructA()
}

func ExpectStructA(count int) []*StructA {
	var s []*StructA
	for i := range count {
		s = append(s, &StructA{Tag: i})
	}
	return s
}

func ExpectInterfaceA(count int) []InterfaceA {
	var s []InterfaceA
	for i := range count {
		s = append(s, &StructA{Tag: i})
	}
	return s
}
