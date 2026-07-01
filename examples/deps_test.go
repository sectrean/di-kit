package main

import (
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/ditest"
	"github.com/sectrean/di-kit/examples/service"
	"github.com/stretchr/testify/require"
)

func Test_Deps(t *testing.T) {
	c, err := di.NewContainer(
		di.WithModule(Deps),
		di.WithDependencyValidation(),
	)
	require.NoError(t, err)

	// Make sure the root service and its dependencies are registered in the container
	ditest.AssertContains[*service.Service](t, c)
}
