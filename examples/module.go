package main

import (
	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/examples/foo"
)

var DependencyModule = di.Module{
	di.WithService(NewLogger),
	di.WithService(foo.NewFooService),
}
