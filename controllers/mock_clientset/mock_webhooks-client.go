// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/coralogix/coralogix-operator/controllers/clientset (interfaces: WebhooksClientInterface)
//
// Generated by this command:
//
//	mockgen -destination=../mock_clientset/mock_webhooks-client.go -package=mock_clientset github.com/coralogix/coralogix-operator/controllers/clientset WebhooksClientInterface
//
// Package mock_clientset is a generated GoMock package.
package mock_clientset

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockWebhooksClientInterface is a mock of WebhooksClientInterface interface.
type MockWebhooksClientInterface struct {
	ctrl     *gomock.Controller
	recorder *MockWebhooksClientInterfaceMockRecorder
}

// MockWebhooksClientInterfaceMockRecorder is the mock recorder for MockWebhooksClientInterface.
type MockWebhooksClientInterfaceMockRecorder struct {
	mock *MockWebhooksClientInterface
}

// NewMockWebhooksClientInterface creates a new mock instance.
func NewMockWebhooksClientInterface(ctrl *gomock.Controller) *MockWebhooksClientInterface {
	mock := &MockWebhooksClientInterface{ctrl: ctrl}
	mock.recorder = &MockWebhooksClientInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWebhooksClientInterface) EXPECT() *MockWebhooksClientInterfaceMockRecorder {
	return m.recorder
}

// CreateWebhook mocks base method.
func (m *MockWebhooksClientInterface) CreateWebhook(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateWebhook", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateWebhook indicates an expected call of CreateWebhook.
func (mr *MockWebhooksClientInterfaceMockRecorder) CreateWebhook(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateWebhook", reflect.TypeOf((*MockWebhooksClientInterface)(nil).CreateWebhook), arg0, arg1)
}

// DeleteWebhook mocks base method.
func (m *MockWebhooksClientInterface) DeleteWebhook(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteWebhook", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteWebhook indicates an expected call of DeleteWebhook.
func (mr *MockWebhooksClientInterfaceMockRecorder) DeleteWebhook(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteWebhook", reflect.TypeOf((*MockWebhooksClientInterface)(nil).DeleteWebhook), arg0, arg1)
}

// GetWebhook mocks base method.
func (m *MockWebhooksClientInterface) GetWebhook(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWebhook", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWebhook indicates an expected call of GetWebhook.
func (mr *MockWebhooksClientInterfaceMockRecorder) GetWebhook(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWebhook", reflect.TypeOf((*MockWebhooksClientInterface)(nil).GetWebhook), arg0, arg1)
}

// GetWebhooks mocks base method.
func (m *MockWebhooksClientInterface) GetWebhooks(arg0 context.Context) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWebhooks", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWebhooks indicates an expected call of GetWebhooks.
func (mr *MockWebhooksClientInterfaceMockRecorder) GetWebhooks(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWebhooks", reflect.TypeOf((*MockWebhooksClientInterface)(nil).GetWebhooks), arg0)
}

// UpdateWebhook mocks base method.
func (m *MockWebhooksClientInterface) UpdateWebhook(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateWebhook", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateWebhook indicates an expected call of UpdateWebhook.
func (mr *MockWebhooksClientInterfaceMockRecorder) UpdateWebhook(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateWebhook", reflect.TypeOf((*MockWebhooksClientInterface)(nil).UpdateWebhook), arg0, arg1)
}
