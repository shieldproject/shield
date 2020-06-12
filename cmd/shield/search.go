package main

import (
	"os"

	"github.com/jhunt/go-table"

	"github.com/shieldproject/shield/client/v2/shield"
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

func SearchBuckets(c *shield.Client, q string) {
	l, err := c.FindBuckets(q, true)
	bail(err)

	tbl := table.NewTable("Key", "Name", "Compression", "Encryption")
	for _, x := range l {
		tbl.Row(x, x.Key, x.Name, x.Compression, x.Encryption)
	}
	tbl.Output(os.Stderr)
}
