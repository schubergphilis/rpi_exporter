// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	mock "github.com/stretchr/testify/mock"
)

// NewExecutor creates a new instance of Executor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewExecutor(t interface {
	mock.TestingT
	Cleanup(func())
}) *Executor {
	mock := &Executor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Executor is an autogenerated mock type for the Executor type
type Executor struct {
	mock.Mock
}

type Executor_Expecter struct {
	mock *mock.Mock
}

func (_m *Executor) EXPECT() *Executor_Expecter {
	return &Executor_Expecter{mock: &_m.Mock}
}

// Run provides a mock function for the type Executor
func (_mock *Executor) Run() {
	_mock.Called()
	return
}

// Executor_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type Executor_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
func (_e *Executor_Expecter) Run() *Executor_Run_Call {
	return &Executor_Run_Call{Call: _e.mock.On("Run")}
}

func (_c *Executor_Run_Call) Run(run func()) *Executor_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Executor_Run_Call) Return() *Executor_Run_Call {
	_c.Call.Return()
	return _c
}

func (_c *Executor_Run_Call) RunAndReturn(run func()) *Executor_Run_Call {
	_c.Run(run)
	return _c
}
