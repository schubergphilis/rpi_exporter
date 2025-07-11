// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	mock "github.com/stretchr/testify/mock"
)

// NewPresenter creates a new instance of Presenter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPresenter(t interface {
	mock.TestingT
	Cleanup(func())
}) *Presenter {
	mock := &Presenter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Presenter is an autogenerated mock type for the Presenter type
type Presenter struct {
	mock.Mock
}

type Presenter_Expecter struct {
	mock *mock.Mock
}

func (_m *Presenter) EXPECT() *Presenter_Expecter {
	return &Presenter_Expecter{mock: &_m.Mock}
}

// Run provides a mock function for the type Presenter
func (_mock *Presenter) Run() {
	_mock.Called()
	return
}

// Presenter_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type Presenter_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
func (_e *Presenter_Expecter) Run() *Presenter_Run_Call {
	return &Presenter_Run_Call{Call: _e.mock.On("Run")}
}

func (_c *Presenter_Run_Call) Run(run func()) *Presenter_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Presenter_Run_Call) Return() *Presenter_Run_Call {
	_c.Call.Return()
	return _c
}

func (_c *Presenter_Run_Call) RunAndReturn(run func()) *Presenter_Run_Call {
	_c.Run(run)
	return _c
}
