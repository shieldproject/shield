package ui

import (
	"fmt"
)

type indentingUI struct {
	parent UI
}

func NewIndentingUI(parent UI) UI {
	return &indentingUI{
		parent: parent,
	}
}

func (ui *indentingUI) ErrorLinef(pattern string, args ...interface{}) {
	ui.parent.ErrorLinef(fmt.Sprintf("  %s", fmt.Sprintf(pattern, args...)))
}

func (ui *indentingUI) PrintLinef(pattern string, args ...interface{}) {
	ui.parent.PrintLinef(fmt.Sprintf("  %s", fmt.Sprintf(pattern, args...)))
}

func (ui *indentingUI) BeginLinef(pattern string, args ...interface{}) {
	ui.parent.BeginLinef(fmt.Sprintf("  %s", fmt.Sprintf(pattern, args...)))
}

func (ui *indentingUI) EndLinef(pattern string, args ...interface{}) {
	ui.parent.EndLinef(fmt.Sprintf(pattern, args...))
}
