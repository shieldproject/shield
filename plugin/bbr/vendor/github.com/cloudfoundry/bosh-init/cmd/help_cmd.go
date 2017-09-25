package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	biui "github.com/cloudfoundry/bosh-init/ui"
)

type helpCmd struct {
	ui          biui.UI
	commandList CommandList
}

func NewHelpCmd(ui biui.UI, commandList CommandList) Cmd {
	return &helpCmd{
		ui:          ui,
		commandList: commandList,
	}
}

func (h *helpCmd) Name() string {
	return "help"
}

func (h *helpCmd) Meta() Meta {
	return Meta{
		Synopsis: "Show help message",
		Usage:    "[command]",
	}
}

func (h *helpCmd) Run(_ biui.Stage, args []string) error {
	if len(args) == 0 {
		h.printGeneral()
		return nil
	}

	cmd, err := h.commandList.Create(args[0])
	if err != nil {
		h.printMissing(args[0])
		return nil
	}

	h.printCommand(cmd.Name(), cmd.Meta())
	return nil
}

func (h *helpCmd) printGeneral() {
	context := helpContext{
		Name:     "bosh-init",
		Synopsis: "A command line tool to initialize BOSH deployments",
		Usage:    "<command> [arguments...]",
		Commands: h.sortedCommands(),
	}

	h.ui.PrintLinef(context.Render())
}

func (h *helpCmd) printCommand(name string, meta Meta) {
	context := helpContext{
		Name:         name,
		IsSubcommand: true,
		Synopsis:     meta.Synopsis,
		Usage:        meta.Usage,
		Envs:         sortedEnvs(meta.Env),
	}

	h.ui.PrintLinef(context.Render())
}

func (h *helpCmd) printMissing(cmdName string) {
	message := "No help found for command `%s'. Run 'bosh-init help' to see all available commands."
	h.ui.PrintLinef(fmt.Sprintf(message, cmdName))
}

func (h *helpCmd) sortedCommands() []contextPair {
	inputs := map[string]string{}
	for key := range h.commandList {
		cmd, _ := h.commandList.Create(key)
		inputs[key] = cmd.Meta().Synopsis
	}

	return sortedPairs(inputs)
}

func sortedEnvs(metaEnvs map[string]MetaEnv) []contextPair {
	inputs := map[string]string{}
	var key string
	for name, env := range metaEnvs {
		if env.Example != "" {
			key = fmt.Sprintf("%s=%s", name, env.Example)
		} else {
			key = name
		}

		if env.Default != "" {
			inputs[key] = fmt.Sprintf("%s. Default: %s", env.Description, env.Default)
		} else {
			inputs[key] = env.Description
		}
	}

	return sortedPairs(inputs)
}

func sortedPairs(pairs map[string]string) []contextPair {
	keys := make([]string, 0, len(pairs))
	maxLen := 0

	for key := range pairs {
		if len(key) > maxLen {
			maxLen = len(key)
		}
		keys = append(keys, key)
	}

	sort.Strings(keys)

	contextPairs := []contextPair{}

	for _, key := range keys {
		value := pairs[key]
		key = fmt.Sprintf("    %s    %s", key, strings.Repeat(" ", maxLen-len(key)))
		contextPairs = append(contextPairs, contextPair{
			Key:   key,
			Value: value,
		})
	}

	return contextPairs
}

const helpTemplate = `NAME:
    {{.Name}} - {{.Synopsis}}

USAGE:
    bosh-init [global options]{{ if .IsSubcommand }} {{.Name}}{{ end }}{{ if .Usage }} {{ .Usage }}{{ end }}{{ if .Commands }}

COMMANDS:{{ range .Commands }}
{{ .Key }}{{ .Value }}{{ end }}{{ end }}{{ if .Envs }}

ENVIRONMENT VARIABLES:{{ range .Envs }}
{{ .Key }}{{ .Value }}{{ end }}{{ end }}

GLOBAL OPTIONS:
    --help, -h       Show help message
    --version, -v    Show version`

type helpContext struct {
	Name         string
	IsSubcommand bool
	Synopsis     string
	Usage        string
	Commands     []contextPair
	Envs         []contextPair
}

func (c *helpContext) Render() string {
	buffer := bytes.NewBuffer([]byte{})
	t := template.Must(template.New("help").Parse(helpTemplate))
	err := t.Execute(buffer, c)
	if err != nil {
		fmt.Printf("Error printing help: %s\n", err.Error())
	}
	return strings.TrimRight(buffer.String(), "\r\n")
}

type contextPair struct {
	Key   string
	Value string
}
