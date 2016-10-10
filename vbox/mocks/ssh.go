// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/vbox (interfaces: SSH)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
	io "io"
	time "time"
)

// Mock of SSH interface
type MockSSH struct {
	ctrl     *gomock.Controller
	recorder *_MockSSHRecorder
}

// Recorder for MockSSH (not exported)
type _MockSSHRecorder struct {
	mock *MockSSH
}

func NewMockSSH(ctrl *gomock.Controller) *MockSSH {
	mock := &MockSSH{ctrl: ctrl}
	mock.recorder = &_MockSSHRecorder{mock}
	return mock
}

func (_m *MockSSH) EXPECT() *_MockSSHRecorder {
	return _m.recorder
}

func (_m *MockSSH) GenerateAddress() (string, string, error) {
	ret := _m.ctrl.Call(_m, "GenerateAddress")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (_mr *_MockSSHRecorder) GenerateAddress() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GenerateAddress")
}

func (_m *MockSSH) GenerateKeypair() ([]byte, []byte, error) {
	ret := _m.ctrl.Call(_m, "GenerateKeypair")
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].([]byte)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (_mr *_MockSSHRecorder) GenerateKeypair() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GenerateKeypair")
}

func (_m *MockSSH) RunSSHCommand(_param0 string, _param1 string, _param2 string, _param3 []byte, _param4 time.Duration, _param5 io.Writer, _param6 io.Writer) error {
	ret := _m.ctrl.Call(_m, "RunSSHCommand", _param0, _param1, _param2, _param3, _param4, _param5, _param6)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockSSHRecorder) RunSSHCommand(arg0, arg1, arg2, arg3, arg4, arg5, arg6 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "RunSSHCommand", arg0, arg1, arg2, arg3, arg4, arg5, arg6)
}
