package main

import (
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/stretchr/testify/assert"
)

func Test_Deps(t *testing.T) {
	_, err := di.NewContainer(
		di.WithModule(Deps),
		di.WithDependencyValidation(),
	)

	assert.NoError(t, err, "NewContainer should be successful")
}
