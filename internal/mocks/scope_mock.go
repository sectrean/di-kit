// Code generated by mockery v2.43.0. DO NOT EDIT.

package mocks

import (
	context "context"

	di "github.com/johnrutherford/di-kit"
	mock "github.com/stretchr/testify/mock"

	reflect "reflect"
)

// ScopeMock is an autogenerated mock type for the Scope type
type ScopeMock struct {
	mock.Mock
}

type ScopeMock_Expecter struct {
	mock *mock.Mock
}

func (_m *ScopeMock) EXPECT() *ScopeMock_Expecter {
	return &ScopeMock_Expecter{mock: &_m.Mock}
}

// Contains provides a mock function with given fields: t, opts
func (_m *ScopeMock) Contains(t reflect.Type, opts ...di.ResolveOption) bool {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, t)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Contains")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func(reflect.Type, ...di.ResolveOption) bool); ok {
		r0 = rf(t, opts...)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// ScopeMock_Contains_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Contains'
type ScopeMock_Contains_Call struct {
	*mock.Call
}

// Contains is a helper method to define mock.On call
//   - t reflect.Type
//   - opts ...di.ResolveOption
func (_e *ScopeMock_Expecter) Contains(t interface{}, opts ...interface{}) *ScopeMock_Contains_Call {
	return &ScopeMock_Contains_Call{Call: _e.mock.On("Contains",
		append([]interface{}{t}, opts...)...)}
}

func (_c *ScopeMock_Contains_Call) Run(run func(t reflect.Type, opts ...di.ResolveOption)) *ScopeMock_Contains_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]di.ResolveOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(di.ResolveOption)
			}
		}
		run(args[0].(reflect.Type), variadicArgs...)
	})
	return _c
}

func (_c *ScopeMock_Contains_Call) Return(_a0 bool) *ScopeMock_Contains_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ScopeMock_Contains_Call) RunAndReturn(run func(reflect.Type, ...di.ResolveOption) bool) *ScopeMock_Contains_Call {
	_c.Call.Return(run)
	return _c
}

// Resolve provides a mock function with given fields: ctx, t, opts
func (_m *ScopeMock) Resolve(ctx context.Context, t reflect.Type, opts ...di.ResolveOption) (interface{}, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, t)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Resolve")
	}

	var r0 interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, reflect.Type, ...di.ResolveOption) (interface{}, error)); ok {
		return rf(ctx, t, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, reflect.Type, ...di.ResolveOption) interface{}); ok {
		r0 = rf(ctx, t, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, reflect.Type, ...di.ResolveOption) error); ok {
		r1 = rf(ctx, t, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ScopeMock_Resolve_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Resolve'
type ScopeMock_Resolve_Call struct {
	*mock.Call
}

// Resolve is a helper method to define mock.On call
//   - ctx context.Context
//   - t reflect.Type
//   - opts ...di.ResolveOption
func (_e *ScopeMock_Expecter) Resolve(ctx interface{}, t interface{}, opts ...interface{}) *ScopeMock_Resolve_Call {
	return &ScopeMock_Resolve_Call{Call: _e.mock.On("Resolve",
		append([]interface{}{ctx, t}, opts...)...)}
}

func (_c *ScopeMock_Resolve_Call) Run(run func(ctx context.Context, t reflect.Type, opts ...di.ResolveOption)) *ScopeMock_Resolve_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]di.ResolveOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(di.ResolveOption)
			}
		}
		run(args[0].(context.Context), args[1].(reflect.Type), variadicArgs...)
	})
	return _c
}

func (_c *ScopeMock_Resolve_Call) Return(_a0 interface{}, _a1 error) *ScopeMock_Resolve_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ScopeMock_Resolve_Call) RunAndReturn(run func(context.Context, reflect.Type, ...di.ResolveOption) (interface{}, error)) *ScopeMock_Resolve_Call {
	_c.Call.Return(run)
	return _c
}

// NewScopeMock creates a new instance of ScopeMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewScopeMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *ScopeMock {
	mock := &ScopeMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
