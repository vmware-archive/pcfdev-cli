// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/pivotal-cf/pcfdev-cli/plugin/cmd (interfaces: VMBuilder)

package mocks

import (
	gomock "github.com/golang/mock/gomock"
	vm "github.com/pivotal-cf/pcfdev-cli/vm"
)

// Mock of VMBuilder interface
type MockVMBuilder struct {
	ctrl     *gomock.Controller
	recorder *_MockVMBuilderRecorder
}

// Recorder for MockVMBuilder (not exported)
type _MockVMBuilderRecorder struct {
	mock *MockVMBuilder
}

func NewMockVMBuilder(ctrl *gomock.Controller) *MockVMBuilder {
	mock := &MockVMBuilder{ctrl: ctrl}
	mock.recorder = &_MockVMBuilderRecorder{mock}
	return mock
}

func (_m *MockVMBuilder) EXPECT() *_MockVMBuilderRecorder {
	return _m.recorder
}

func (_m *MockVMBuilder) VM(_param0 string) (vm.VM, error) {
	ret := _m.ctrl.Call(_m, "VM", _param0)
	ret0, _ := ret[0].(vm.VM)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockVMBuilderRecorder) VM(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "VM", arg0)
}
