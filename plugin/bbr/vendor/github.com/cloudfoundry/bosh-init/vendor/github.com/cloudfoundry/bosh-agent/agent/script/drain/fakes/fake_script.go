package fakes

import (
	"github.com/cloudfoundry/bosh-agent/agent/script/drain"
)

type FakeScript struct {
	tag        string
	Params     drain.ScriptParams
	ExistsBool bool

	RunCallCount int
	DidRun       bool
	RunError     error
	RunStub      func() error
	WasCanceled  bool
}

func NewFakeScript(tag string) *FakeScript {
	return &FakeScript{tag: tag, ExistsBool: true}
}

func (s *FakeScript) Tag() string  { return s.tag }
func (s *FakeScript) Path() string { return "/fake/path" }
func (s *FakeScript) Exists() bool { return s.ExistsBool }

func (s *FakeScript) Cancel() error {
	s.WasCanceled = true
	return nil
}

func (s *FakeScript) Run() error {
	s.DidRun = true
	s.RunCallCount++
	if s.RunStub != nil {
		return s.RunStub()
	}
	return s.RunError
}
