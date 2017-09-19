// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/cloudfoundry/bosh-cli/installation (interfaces: Installation,Installer,InstallerFactory,Uninstaller,JobResolver,PackageCompiler,JobRenderer)

package mocks

import (
	installation "github.com/cloudfoundry/bosh-cli/installation"
	manifest "github.com/cloudfoundry/bosh-cli/installation/manifest"
	job "github.com/cloudfoundry/bosh-cli/release/job"
	ui "github.com/cloudfoundry/bosh-cli/ui"
	logger "github.com/cloudfoundry/bosh-utils/logger"
	gomock "github.com/golang/mock/gomock"
)

// Mock of Installation interface
type MockInstallation struct {
	ctrl     *gomock.Controller
	recorder *_MockInstallationRecorder
}

// Recorder for MockInstallation (not exported)
type _MockInstallationRecorder struct {
	mock *MockInstallation
}

func NewMockInstallation(ctrl *gomock.Controller) *MockInstallation {
	mock := &MockInstallation{ctrl: ctrl}
	mock.recorder = &_MockInstallationRecorder{mock}
	return mock
}

func (_m *MockInstallation) EXPECT() *_MockInstallationRecorder {
	return _m.recorder
}

func (_m *MockInstallation) Job() installation.InstalledJob {
	ret := _m.ctrl.Call(_m, "Job")
	ret0, _ := ret[0].(installation.InstalledJob)
	return ret0
}

func (_mr *_MockInstallationRecorder) Job() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Job")
}

func (_m *MockInstallation) StartRegistry() error {
	ret := _m.ctrl.Call(_m, "StartRegistry")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockInstallationRecorder) StartRegistry() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "StartRegistry")
}

func (_m *MockInstallation) StopRegistry() error {
	ret := _m.ctrl.Call(_m, "StopRegistry")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockInstallationRecorder) StopRegistry() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "StopRegistry")
}

func (_m *MockInstallation) Target() installation.Target {
	ret := _m.ctrl.Call(_m, "Target")
	ret0, _ := ret[0].(installation.Target)
	return ret0
}

func (_mr *_MockInstallationRecorder) Target() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Target")
}

func (_m *MockInstallation) WithRunningRegistry(_param0 logger.Logger, _param1 ui.Stage, _param2 func() error) error {
	ret := _m.ctrl.Call(_m, "WithRunningRegistry", _param0, _param1, _param2)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockInstallationRecorder) WithRunningRegistry(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "WithRunningRegistry", arg0, arg1, arg2)
}

// Mock of Installer interface
type MockInstaller struct {
	ctrl     *gomock.Controller
	recorder *_MockInstallerRecorder
}

// Recorder for MockInstaller (not exported)
type _MockInstallerRecorder struct {
	mock *MockInstaller
}

func NewMockInstaller(ctrl *gomock.Controller) *MockInstaller {
	mock := &MockInstaller{ctrl: ctrl}
	mock.recorder = &_MockInstallerRecorder{mock}
	return mock
}

func (_m *MockInstaller) EXPECT() *_MockInstallerRecorder {
	return _m.recorder
}

func (_m *MockInstaller) Cleanup(_param0 installation.Installation) error {
	ret := _m.ctrl.Call(_m, "Cleanup", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockInstallerRecorder) Cleanup(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Cleanup", arg0)
}

func (_m *MockInstaller) Install(_param0 manifest.Manifest, _param1 ui.Stage) (installation.Installation, error) {
	ret := _m.ctrl.Call(_m, "Install", _param0, _param1)
	ret0, _ := ret[0].(installation.Installation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockInstallerRecorder) Install(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Install", arg0, arg1)
}

// Mock of InstallerFactory interface
type MockInstallerFactory struct {
	ctrl     *gomock.Controller
	recorder *_MockInstallerFactoryRecorder
}

// Recorder for MockInstallerFactory (not exported)
type _MockInstallerFactoryRecorder struct {
	mock *MockInstallerFactory
}

func NewMockInstallerFactory(ctrl *gomock.Controller) *MockInstallerFactory {
	mock := &MockInstallerFactory{ctrl: ctrl}
	mock.recorder = &_MockInstallerFactoryRecorder{mock}
	return mock
}

func (_m *MockInstallerFactory) EXPECT() *_MockInstallerFactoryRecorder {
	return _m.recorder
}

func (_m *MockInstallerFactory) NewInstaller(_param0 installation.Target) installation.Installer {
	ret := _m.ctrl.Call(_m, "NewInstaller", _param0)
	ret0, _ := ret[0].(installation.Installer)
	return ret0
}

func (_mr *_MockInstallerFactoryRecorder) NewInstaller(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "NewInstaller", arg0)
}

// Mock of Uninstaller interface
type MockUninstaller struct {
	ctrl     *gomock.Controller
	recorder *_MockUninstallerRecorder
}

// Recorder for MockUninstaller (not exported)
type _MockUninstallerRecorder struct {
	mock *MockUninstaller
}

func NewMockUninstaller(ctrl *gomock.Controller) *MockUninstaller {
	mock := &MockUninstaller{ctrl: ctrl}
	mock.recorder = &_MockUninstallerRecorder{mock}
	return mock
}

func (_m *MockUninstaller) EXPECT() *_MockUninstallerRecorder {
	return _m.recorder
}

func (_m *MockUninstaller) Uninstall(_param0 installation.Target) error {
	ret := _m.ctrl.Call(_m, "Uninstall", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockUninstallerRecorder) Uninstall(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Uninstall", arg0)
}

// Mock of JobResolver interface
type MockJobResolver struct {
	ctrl     *gomock.Controller
	recorder *_MockJobResolverRecorder
}

// Recorder for MockJobResolver (not exported)
type _MockJobResolverRecorder struct {
	mock *MockJobResolver
}

func NewMockJobResolver(ctrl *gomock.Controller) *MockJobResolver {
	mock := &MockJobResolver{ctrl: ctrl}
	mock.recorder = &_MockJobResolverRecorder{mock}
	return mock
}

func (_m *MockJobResolver) EXPECT() *_MockJobResolverRecorder {
	return _m.recorder
}

func (_m *MockJobResolver) From(_param0 manifest.Manifest) ([]job.Job, error) {
	ret := _m.ctrl.Call(_m, "From", _param0)
	ret0, _ := ret[0].([]job.Job)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockJobResolverRecorder) From(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "From", arg0)
}

// Mock of PackageCompiler interface
type MockPackageCompiler struct {
	ctrl     *gomock.Controller
	recorder *_MockPackageCompilerRecorder
}

// Recorder for MockPackageCompiler (not exported)
type _MockPackageCompilerRecorder struct {
	mock *MockPackageCompiler
}

func NewMockPackageCompiler(ctrl *gomock.Controller) *MockPackageCompiler {
	mock := &MockPackageCompiler{ctrl: ctrl}
	mock.recorder = &_MockPackageCompilerRecorder{mock}
	return mock
}

func (_m *MockPackageCompiler) EXPECT() *_MockPackageCompilerRecorder {
	return _m.recorder
}

func (_m *MockPackageCompiler) For(_param0 []job.Job, _param1 ui.Stage) ([]installation.CompiledPackageRef, error) {
	ret := _m.ctrl.Call(_m, "For", _param0, _param1)
	ret0, _ := ret[0].([]installation.CompiledPackageRef)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockPackageCompilerRecorder) For(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "For", arg0, arg1)
}

// Mock of JobRenderer interface
type MockJobRenderer struct {
	ctrl     *gomock.Controller
	recorder *_MockJobRendererRecorder
}

// Recorder for MockJobRenderer (not exported)
type _MockJobRendererRecorder struct {
	mock *MockJobRenderer
}

func NewMockJobRenderer(ctrl *gomock.Controller) *MockJobRenderer {
	mock := &MockJobRenderer{ctrl: ctrl}
	mock.recorder = &_MockJobRendererRecorder{mock}
	return mock
}

func (_m *MockJobRenderer) EXPECT() *_MockJobRendererRecorder {
	return _m.recorder
}

func (_m *MockJobRenderer) RenderAndUploadFrom(_param0 manifest.Manifest, _param1 []job.Job, _param2 ui.Stage) ([]installation.RenderedJobRef, error) {
	ret := _m.ctrl.Call(_m, "RenderAndUploadFrom", _param0, _param1, _param2)
	ret0, _ := ret[0].([]installation.RenderedJobRef)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockJobRendererRecorder) RenderAndUploadFrom(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "RenderAndUploadFrom", arg0, arg1, arg2)
}
