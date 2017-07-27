package cmd

import (
	"fmt"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshui "github.com/cloudfoundry/bosh-cli/ui"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

type InstancesCmd struct {
	ui       boshui.UI
	director boshdir.Director
}

func NewInstancesCmd(ui boshui.UI, director boshdir.Director) InstancesCmd {
	return InstancesCmd{ui: ui, director: director}
}

func (c InstancesCmd) Run(opts InstancesOpts) error {
	instTable := InstanceTable{
		Processes: opts.Processes,
		Details:   opts.Details,
		DNS:       opts.DNS,
		Vitals:    opts.Vitals,
	}

	if len(opts.Deployment) > 0 {
		dep, err := c.director.FindDeployment(opts.Deployment)
		if err != nil {
			return err
		}

		return c.printDeployment(dep, instTable, opts)
	}

	return c.printDeployments(instTable, opts)
}

func (c InstancesCmd) printDeployments(instTable InstanceTable, opts InstancesOpts) error {
	deployments, err := c.director.Deployments()
	if err != nil {
		return err
	}

	for _, dep := range deployments {
		err := c.printDeployment(dep, instTable, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c InstancesCmd) printDeployment(dep boshdir.Deployment, instTable InstanceTable, opts InstancesOpts) error {
	instanceInfos, err := dep.InstanceInfos()
	if err != nil {
		return err
	}

	table := boshtbl.Table{
		Title: fmt.Sprintf("Deployment '%s'", dep.Name()),

		Content: "instances",

		Header: instTable.Headers(),

		SortBy: []boshtbl.ColumnSort{
			{Column: 0, Asc: true},
			{Column: 1, Asc: true}, // sort by process so that VM row is first
		},
	}

	for _, info := range instanceInfos {
		if opts.Failing && info.IsRunning() {
			continue
		}

		row := instTable.AsValues(instTable.ForVMInfo(info))

		section := boshtbl.Section{
			FirstColumn: row[0],
			Rows:        [][]boshtbl.Value{row},
		}

		if opts.Processes {
			for _, p := range info.Processes {
				if opts.Failing && p.IsRunning() {
					continue
				}

				row := instTable.AsValues(instTable.ForProcess(p))

				section.Rows = append(section.Rows, row)
			}
		}

		table.Sections = append(table.Sections, section)
	}

	c.ui.PrintTable(table)

	return nil
}
