// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/plugin (interfaces: UI)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
)

// Mock of UI interface
type MockUI struct {
	ctrl     *gomock.Controller
	recorder *_MockUIRecorder
}

// Recorder for MockUI (not exported)
type _MockUIRecorder struct {
	mock *MockUI
}

func NewMockUI(ctrl *gomock.Controller) *MockUI {
	mock := &MockUI{ctrl: ctrl}
	mock.recorder = &_MockUIRecorder{mock}
	return mock
}

func (_m *MockUI) EXPECT() *_MockUIRecorder {
	return _m.recorder
}

func (_m *MockUI) Ask(_param0 string) string {
	ret := _m.ctrl.Call(_m, "Ask", _param0)
	ret0, _ := ret[0].(string)
	return ret0
}

func (_mr *_MockUIRecorder) Ask(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Ask", arg0)
}

func (_m *MockUI) Failed(_param0 string, _param1 ...interface{}) {
	_s := []interface{}{_param0}
	for _, _x := range _param1 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Failed", _s...)
}

func (_mr *_MockUIRecorder) Failed(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0}, arg1...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Failed", _s...)
}

func (_m *MockUI) Say(_param0 string, _param1 ...interface{}) {
	_s := []interface{}{_param0}
	for _, _x := range _param1 {
		_s = append(_s, _x)
	}
	_m.ctrl.Call(_m, "Say", _s...)
}

func (_mr *_MockUIRecorder) Say(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0}, arg1...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Say", _s...)
}
