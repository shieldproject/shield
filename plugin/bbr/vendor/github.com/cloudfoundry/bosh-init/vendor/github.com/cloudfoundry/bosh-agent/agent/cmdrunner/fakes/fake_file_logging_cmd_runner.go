package fakes

import (
	boshcmdrunner "github.com/cloudfoundry/bosh-agent/agent/cmdrunner"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type FakeFileLoggingCmdRunner struct {
	RunCommands        []boshsys.Command
	RunCommandJobName  string
	RunCommandTaskName string
	RunCommandResult   *boshcmdrunner.CmdResult
	RunCommandErr      error
}

func NewFakeFileLoggingCmdRunner() *FakeFileLoggingCmdRunner {
	return &FakeFileLoggingCmdRunner{}
}

func (f *FakeFileLoggingCmdRunner) RunCommand(jobName, taskName string, cmd boshsys.Command) (*boshcmdrunner.CmdResult, error) {
	f.RunCommandJobName = jobName
	f.RunCommandTaskName = taskName
	f.RunCommands = append(f.RunCommands, cmd)
	return f.RunCommandResult, f.RunCommandErr
}
