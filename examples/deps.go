package main

import (
	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/examples/service"

	"github.com/sectrean/di-kit/examples/storage"
)

var Deps = di.Module{
	storage.Dependencies,
	di.WithService(NewLogger),
	di.WithService(service.NewService),
}
