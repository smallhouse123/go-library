// Code generated by mockery v2.53.4. DO NOT EDIT.

package mocks

import (
	metrics "github.com/smallhouse123/go-library/service/metrics"
	mock "github.com/stretchr/testify/mock"
)

// Metrics is an autogenerated mock type for the Metrics type
type Metrics struct {
	mock.Mock
}

// BumpCount provides a mock function with given fields: key, val, tags
func (_m *Metrics) BumpCount(key string, val float64, tags ...string) error {
	_va := make([]interface{}, len(tags))
	for _i := range tags {
		_va[_i] = tags[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key, val)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for BumpCount")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, float64, ...string) error); ok {
		r0 = rf(key, val, tags...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BumpTime provides a mock function with given fields: key, tags
func (_m *Metrics) BumpTime(key string, tags ...string) (metrics.Endable, error) {
	_va := make([]interface{}, len(tags))
	for _i := range tags {
		_va[_i] = tags[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, key)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for BumpTime")
	}

	var r0 metrics.Endable
	var r1 error
	if rf, ok := ret.Get(0).(func(string, ...string) (metrics.Endable, error)); ok {
		return rf(key, tags...)
	}
	if rf, ok := ret.Get(0).(func(string, ...string) metrics.Endable); ok {
		r0 = rf(key, tags...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metrics.Endable)
		}
	}

	if rf, ok := ret.Get(1).(func(string, ...string) error); ok {
		r1 = rf(key, tags...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMetrics creates a new instance of Metrics. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMetrics(t interface {
	mock.TestingT
	Cleanup(func())
}) *Metrics {
	mock := &Metrics{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
