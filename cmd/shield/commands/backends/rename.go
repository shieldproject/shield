package backends

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Rename - Rename SHIELD backend alias
var Rename = &commands.Command{
	Summary: "Rename SHIELD backend alias",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "oldname", Mandatory: true, Positional: true,
				Desc: `The currentname for the backend`,
			},
			commands.FlagInfo{
				Name: "newname", Mandatory: true, Positional: true,
				Desc: `The new name for the backend`,
			},
		},
	},
	RunFn: cliRenameBackend,
	Group: commands.BackendsGroup,
}

func cliRenameBackend(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'rename-backend' command")

	if len(args) != 2 {
		return fmt.Errorf("Invalid 'rename-backend' syntax: `shield rename-backend <oldname> <newname>")
	}

	oldname := args[0]
	newname := args[1]

	toRename := config.Get(oldname)
	if toRename == nil {
		return fmt.Errorf("No backend with name `%s' exists", oldname)
	}

	if config.Get(newname) != nil {
		return fmt.Errorf("Backend with name `%s' already exists", newname)
	}

	editingCurrent := config.Current() != nil && config.Current().Name == oldname

	toRename.Name = newname
	err := config.Commit(toRename)
	if err != nil {
		return err
	}

	config.Delete(oldname)

	ansi.Fprintf(os.Stdout, "Successfully renamed backend '@G{%s}' (@G{%s}) to @G{%s}'\n\n", oldname, toRename.Address, newname)

	if editingCurrent {
		config.Use(newname)
		if err != nil {
			panic("We just renamed to this backend... why can't we use it?")
		}
		DisplayCurrent()
	}

	return nil
}
