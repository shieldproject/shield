package cmd

import (
	"errors"
	davclient "github.com/cloudfoundry/bosh-davcli/client"
)

type ExistsCmd struct {
	client davclient.Client
}

func newExistsCmd(client davclient.Client) (cmd ExistsCmd) {
	cmd.client = client
	return
}

func (cmd ExistsCmd) Run(args []string) (err error) {
	if len(args) != 1 {
		err = errors.New("Incorrect usage, exists needs remote blob path")
		return
	}
	err = cmd.client.Exists(args[0])
	return
}
