package main

import (
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/sectrean/di-kit/examples/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Deps(t *testing.T) {
	c, err := di.NewContainer(Deps)
	require.NoError(t, err)

	// Make sure the root service is registered
	assert.True(t, di.Contains[*service.Service](c))

	// Validate that all services are resolvable
	err = di.ValidateContainer(c)
	assert.NoError(t, err)
}
