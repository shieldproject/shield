package fakes

import (
	birel "github.com/cloudfoundry/bosh-init/release"
)

type FakeValidator struct {
	ValidateError error
}

func NewFakeValidator() *FakeValidator {
	return &FakeValidator{}
}

func (f *FakeValidator) Validate(release birel.Release) error {
	return f.ValidateError
}
