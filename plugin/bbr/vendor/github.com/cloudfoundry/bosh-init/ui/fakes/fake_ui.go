package fakes

import (
	"fmt"
)

type FakeUI struct {
	Said   []string
	Errors []string
}

func (ui *FakeUI) ErrorLinef(pattern string, args ...interface{}) {
	ui.Errors = append(ui.Errors, fmt.Sprintf(pattern, args...))
}

func (ui *FakeUI) PrintLinef(pattern string, args ...interface{}) {
	ui.Said = append(ui.Said, fmt.Sprintf(pattern, args...))
}

func (ui *FakeUI) BeginLinef(pattern string, args ...interface{}) {
	ui.Said = append(ui.Said, fmt.Sprintf(pattern, args...))
}

func (ui *FakeUI) EndLinef(pattern string, args ...interface{}) {
	ui.Said = append(ui.Said, fmt.Sprintf(pattern, args...))
}
