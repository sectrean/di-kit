package foo

import (
	"context"
	"fmt"
)

type FooService interface {
	DoFoo()
	Close(ctx context.Context) error
}

func NewFooService() FooService {
	fmt.Println("creating FooServiceImpl")
	return &FooServiceImpl{}
}

type FooServiceImpl struct{}

func (s *FooServiceImpl) DoFoo() {
	fmt.Println("FooServiceImpl: doing foo")
}

func (s *FooServiceImpl) Close(ctx context.Context) error {
	fmt.Println("FooServiceImpl: closing")
	return nil
}
