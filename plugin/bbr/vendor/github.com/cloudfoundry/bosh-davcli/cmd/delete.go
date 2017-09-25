package cmd

import (
	"errors"
	davclient "github.com/cloudfoundry/bosh-davcli/client"
)

type DeleteCmd struct {
	client davclient.Client
}

func newDeleteCmd(client davclient.Client) (cmd DeleteCmd) {
	cmd.client = client
	return
}

func (cmd DeleteCmd) Run(args []string) (err error) {
	if len(args) != 1 {
		err = errors.New("Incorrect usage, delete needs remote blob path")
		return
	}
	err = cmd.client.Delete(args[0])
	return
}
