package ditest_test

import (
	"reflect"
	"testing"

	"github.com/sectrean/di-kit/ditest"
	"github.com/sectrean/di-kit/internal/mocks"
	"github.com/sectrean/di-kit/internal/testtypes"
	"github.com/stretchr/testify/assert"
)

func TestAssertContains(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		scope := mocks.NewScopeMock(t)
		scope.EXPECT().
			Contains(reflect.TypeFor[testtypes.StructA]()).
			Return(true).Once()

		mockT := mocks.NewTestingTMock(t)
		mockT.EXPECT().
			Helper().Once()

		got := ditest.AssertContains[testtypes.StructA](mockT, scope)
		assert.True(t, got)
	})

	t.Run("false", func(t *testing.T) {
		scope := mocks.NewScopeMock(t)
		scope.EXPECT().
			Contains(reflect.TypeFor[testtypes.StructB]()).
			Return(false).Once()

		mockT := mocks.NewTestingTMock(t)
		mockT.EXPECT().
			Helper().Once()
		mockT.EXPECT().
			Errorf(
				"ditest.AssertContains: Scope should contain type %s",
				reflect.TypeFor[testtypes.StructB](),
			).Once()

		got := ditest.AssertContains[testtypes.StructB](mockT, scope)
		assert.False(t, got)
	})
}

func TestAssertNotContains(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		scope := mocks.NewScopeMock(t)
		scope.EXPECT().
			Contains(reflect.TypeFor[testtypes.StructB]()).
			Return(false).Once()

		mockT := mocks.NewTestingTMock(t)
		mockT.EXPECT().
			Helper().Once()

		got := ditest.AssertNotContains[testtypes.StructB](mockT, scope)
		assert.True(t, got)
	})

	t.Run("false", func(t *testing.T) {
		scope := mocks.NewScopeMock(t)
		scope.EXPECT().
			Contains(reflect.TypeFor[testtypes.StructA]()).
			Return(true).Once()

		mockT := mocks.NewTestingTMock(t)
		mockT.EXPECT().
			Helper().Once()
		mockT.EXPECT().
			Errorf(
				"ditest.AssertNotContains: Scope should not contain type %s",
				reflect.TypeFor[testtypes.StructA](),
			).Once()

		got := ditest.AssertNotContains[testtypes.StructA](mockT, scope)
		assert.False(t, got)
	})
}
