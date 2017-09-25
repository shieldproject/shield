package drain

import (
	"strconv"
	"strings"
	"time"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	"github.com/pivotal-golang/clock"
)

type ConcreteScript struct {
	fs     boshsys.FileSystem
	runner boshsys.CmdRunner

	tag    string
	path   string
	params ScriptParams

	timeService clock.Clock
	logTag      string
	logger      boshlog.Logger

	cancelCh chan struct{}
}

func NewConcreteScript(
	fs boshsys.FileSystem,
	runner boshsys.CmdRunner,
	tag string,
	path string,
	params ScriptParams,
	timeService clock.Clock,
	logger boshlog.Logger,
) ConcreteScript {
	return ConcreteScript{
		fs:     fs,
		runner: runner,

		tag:    tag,
		path:   path,
		params: params,

		timeService: timeService,

		logTag: "DrainScript",
		logger: logger,

		cancelCh: make(chan struct{}, 1),
	}
}

func (s ConcreteScript) Tag() string          { return s.tag }
func (s ConcreteScript) Path() string         { return s.path }
func (s ConcreteScript) Params() ScriptParams { return s.params }
func (s ConcreteScript) Exists() bool         { return s.fs.FileExists(s.path) }

func (s ConcreteScript) Run() error {
	params := s.params

	for {
		value, err := s.runOnce(params)
		if err != nil {
			return err
		} else if value < 0 {
			s.timeService.Sleep(time.Duration(-value) * time.Second)
			params = params.ToStatusParams()
		} else {
			s.timeService.Sleep(time.Duration(value) * time.Second)
			return nil
		}
	}
}

func (s ConcreteScript) Cancel() error {
	select {
	case s.cancelCh <- struct{}{}:
	default:
	}
	return nil
}

func (s ConcreteScript) runOnce(params ScriptParams) (int, error) {
	jobChange := params.JobChange()
	hashChange := params.HashChange()
	updatedPkgs := params.UpdatedPackages()

	command := boshsys.Command{
		Name: s.path,
		Env: map[string]string{
			"PATH": "/usr/sbin:/usr/bin:/sbin:/bin",
		},
	}

	jobState, err := params.JobState()
	if err != nil {
		return 0, bosherr.WrapError(err, "Getting job state")
	}

	if jobState != "" {
		command.Env["BOSH_JOB_STATE"] = jobState
	}

	jobNextState, err := params.JobNextState()
	if err != nil {
		return 0, bosherr.WrapError(err, "Getting job next state")
	}

	if jobNextState != "" {
		command.Env["BOSH_JOB_NEXT_STATE"] = jobNextState
	}

	command.Args = append(command.Args, jobChange, hashChange)
	command.Args = append(command.Args, updatedPkgs...)

	process, err := s.runner.RunComplexCommandAsync(command)
	if err != nil {
		return 0, bosherr.WrapError(err, "Running drain script")
	}

	var result boshsys.Result

	isCanceled := false

	// Can only wait once on a process but cancelling can happen multiple times
	for processExitedCh := process.Wait(); processExitedCh != nil; {
		select {
		case result = <-processExitedCh:
			processExitedCh = nil
		case <-s.cancelCh:
			// Ignore possible TerminateNicely error since we cannot return it
			err := process.TerminateNicely(10 * time.Second)
			if err != nil {
				s.logger.Error(s.logTag, "Failed to terminate %s", err.Error())
			}
			isCanceled = true
		}
	}

	if isCanceled {
		if result.Error != nil {
			return 0, bosherr.WrapError(result.Error, "Script was cancelled by user request")
		}

		return 0, bosherr.Error("Script was cancelled by user request")
	}

	if result.Error != nil && result.ExitStatus == -1 {
		return 0, bosherr.WrapError(result.Error, "Running drain script")
	}

	value, err := strconv.Atoi(strings.TrimSpace(result.Stdout))
	if err != nil {
		return 0, bosherr.WrapError(err, "Script did not return a signed integer")
	}

	return value, nil
}
