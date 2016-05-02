// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/plugin (interfaces: FS)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
	io "io"
)

// Mock of FS interface
type MockFS struct {
	ctrl     *gomock.Controller
	recorder *_MockFSRecorder
}

// Recorder for MockFS (not exported)
type _MockFSRecorder struct {
	mock *MockFS
}

func NewMockFS(ctrl *gomock.Controller) *MockFS {
	mock := &MockFS{ctrl: ctrl}
	mock.recorder = &_MockFSRecorder{mock}
	return mock
}

func (_m *MockFS) EXPECT() *_MockFSRecorder {
	return _m.recorder
}

func (_m *MockFS) CreateDir(_param0 string) error {
	ret := _m.ctrl.Call(_m, "CreateDir", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockFSRecorder) CreateDir(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CreateDir", arg0)
}

func (_m *MockFS) Exists(_param0 string) (bool, error) {
	ret := _m.ctrl.Call(_m, "Exists", _param0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockFSRecorder) Exists(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Exists", arg0)
}

func (_m *MockFS) MD5(_param0 string) (string, error) {
	ret := _m.ctrl.Call(_m, "MD5", _param0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockFSRecorder) MD5(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "MD5", arg0)
}

func (_m *MockFS) RemoveFile(_param0 string) error {
	ret := _m.ctrl.Call(_m, "RemoveFile", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockFSRecorder) RemoveFile(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "RemoveFile", arg0)
}

func (_m *MockFS) Write(_param0 string, _param1 io.Reader) error {
	ret := _m.ctrl.Call(_m, "Write", _param0, _param1)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockFSRecorder) Write(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Write", arg0, arg1)
}
