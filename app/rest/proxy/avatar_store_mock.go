// Code generated by mockery v1.0.0. DO NOT EDIT.
package proxy

import io "io"
import mock "github.com/stretchr/testify/mock"

// MockAvatarStore is an autogenerated mock type for the AvatarStore type
type MockAvatarStore struct {
	mock.Mock
}

// Get provides a mock function with given fields: avatar
func (_m *MockAvatarStore) Get(avatar string) (io.ReadCloser, int, error) {
	ret := _m.Called(avatar)

	var r0 io.ReadCloser
	if rf, ok := ret.Get(0).(func(string) io.ReadCloser); ok {
		r0 = rf(avatar)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.ReadCloser)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(string) int); ok {
		r1 = rf(avatar)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(avatar)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// Put provides a mock function with given fields: userID, reader
func (_m *MockAvatarStore) Put(userID string, reader io.Reader) (string, error) {
	ret := _m.Called(userID, reader)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, io.Reader) string); ok {
		r0 = rf(userID, reader)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, io.Reader) error); ok {
		r1 = rf(userID, reader)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
