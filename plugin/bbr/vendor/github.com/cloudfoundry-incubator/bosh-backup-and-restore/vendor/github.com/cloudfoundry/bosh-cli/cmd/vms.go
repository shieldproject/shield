package cmd

import (
	"fmt"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshui "github.com/cloudfoundry/bosh-cli/ui"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

type VMsCmd struct {
	ui       boshui.UI
	director boshdir.Director
}

func NewVMsCmd(ui boshui.UI, director boshdir.Director) VMsCmd {
	return VMsCmd{ui: ui, director: director}
}

func (c VMsCmd) Run(opts VMsOpts) error {
	instTable := InstanceTable{
		// VMs command should always show VM specifics
		VMDetails: true,

		Details: false,
		DNS:     opts.DNS,
		Vitals:  opts.Vitals,
	}

	if len(opts.Deployment) > 0 {
		dep, err := c.director.FindDeployment(opts.Deployment)
		if err != nil {
			return err
		}

		return c.printDeployment(dep, instTable)
	}

	return c.printDeployments(instTable)
}

func (c VMsCmd) printDeployments(instTable InstanceTable) error {
	deployments, err := c.director.Deployments()
	if err != nil {
		return err
	}

	for _, dep := range deployments {
		err := c.printDeployment(dep, instTable)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c VMsCmd) printDeployment(dep boshdir.Deployment, instTable InstanceTable) error {
	vmInfos, err := dep.VMInfos()
	if err != nil {
		return err
	}

	table := boshtbl.Table{
		Title: fmt.Sprintf("Deployment '%s'", dep.Name()),

		Content: "vms",

		Header: instTable.Headers(),

		SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},
	}

	for _, info := range vmInfos {
		row := instTable.AsValues(instTable.ForVMInfo(info))

		table.Rows = append(table.Rows, row)
	}

	c.ui.PrintTable(table)

	return nil
}
