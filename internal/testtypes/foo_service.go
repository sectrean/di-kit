package testtypes

// FooService has no dependencies
type FooService interface {
	Foo()
}

func NewFooService() FooService {
	return &MockFooService{}
}
