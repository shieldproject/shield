package fakes

import (
	"fmt"

	. "github.com/cloudfoundry/bosh-cli/ui/table"
)

type FakeUI struct {
	Said   []string
	Errors []string

	Blocks []string

	Table  Table
	Tables []Table

	AskedTextLabels []string
	AskedText       []Answer

	AskedPasswordLabels []string
	AskedPasswords      []Answer

	AskedChoiceCalled  bool
	AskedChoiceLabel   string
	AskedChoiceOptions []string
	AskedChoiceChosens []int
	AskedChoiceErrs    []error

	AskedConfirmationCalled bool
	AskedConfirmationErr    error

	Interactive bool

	Flushed bool
}

type Answer struct {
	Text  string
	Error error
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

func (ui *FakeUI) PrintBlock(block string) {
	ui.Blocks = append(ui.Blocks, block)
}

func (ui *FakeUI) PrintErrorBlock(block string) {
	ui.Blocks = append(ui.Blocks, block)
}

func (ui *FakeUI) PrintTable(table Table) {
	ui.Table = table
	ui.Tables = append(ui.Tables, table)
}

func (ui *FakeUI) AskForText(label string) (string, error) {
	ui.AskedTextLabels = append(ui.AskedTextLabels, label)
	answer := ui.AskedText[0]
	ui.AskedText = ui.AskedText[1:]
	return answer.Text, answer.Error
}

func (ui *FakeUI) AskForChoice(label string, options []string) (int, error) {
	ui.AskedChoiceCalled = true

	ui.AskedChoiceLabel = label
	ui.AskedChoiceOptions = options

	chosen := ui.AskedChoiceChosens[0]
	ui.AskedChoiceChosens = ui.AskedChoiceChosens[1:]

	err := ui.AskedChoiceErrs[0]
	ui.AskedChoiceErrs = ui.AskedChoiceErrs[1:]

	return chosen, err
}

func (ui *FakeUI) AskForPassword(label string) (string, error) {
	ui.AskedPasswordLabels = append(ui.AskedPasswordLabels, label)
	answer := ui.AskedPasswords[0]
	ui.AskedPasswords = ui.AskedPasswords[1:]
	return answer.Text, answer.Error
}

func (ui *FakeUI) AskForConfirmation() error {
	ui.AskedConfirmationCalled = true
	return ui.AskedConfirmationErr
}

func (ui *FakeUI) IsInteractive() bool {
	return ui.Interactive
}

func (ui *FakeUI) Flush() {
	ui.Flushed = true
}
