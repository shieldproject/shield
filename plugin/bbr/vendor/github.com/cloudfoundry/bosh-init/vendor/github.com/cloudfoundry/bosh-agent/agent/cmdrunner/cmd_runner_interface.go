package cmdrunner

import (
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type CmdResult struct {
	IsStdoutTruncated bool
	IsStderrTruncated bool

	// Not using string to avoid copying
	Stdout []byte
	Stderr []byte

	ExitStatus int
}

type CmdRunner interface {
	RunCommand(jobName, taskName string, cmd boshsys.Command) (*CmdResult, error)
}
