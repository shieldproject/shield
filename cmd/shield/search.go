package main

import (
	"os"

	"github.com/jhunt/go-table"

	"github.com/starkandwayne/shield/client/v2/shield"
)

func SearchTargets(c *shield.Client, tenant *shield.Tenant, q string) {
	l, err := c.ListTargets(tenant, &shield.TargetFilter{
		Name:  q,
		Fuzzy: true,
	})
	bail(err)

	tbl := table.NewTable("UUID", "Name", "SHIELD Agent", "Plugin")
	for _, x := range l {
		tbl.Row(x, x.UUID, x.Name, x.Agent, x.Plugin)
	}
	tbl.Output(os.Stderr)
}

func SearchStores(c *shield.Client, tenant *shield.Tenant, q string) {
	l, err := c.ListStores(tenant, &shield.StoreFilter{
		Name:  q,
		Fuzzy: true,
	})
	bail(err)

	g, err := c.ListGlobalStores(&shield.StoreFilter{
		Name:  q,
		Fuzzy: true,
	})
	bail(err)

	tbl := table.NewTable("UUID", "Scope", "Name", "SHIELD Agent", "Plugin")
	for _, x := range g {
		tbl.Row(x, x.UUID, "global", x.Name, x.Agent, x.Plugin)
	}
	for _, x := range l {
		tbl.Row(x, x.UUID, "tenant", x.Name, x.Agent, x.Plugin)
	}
	tbl.Output(os.Stderr)
}
