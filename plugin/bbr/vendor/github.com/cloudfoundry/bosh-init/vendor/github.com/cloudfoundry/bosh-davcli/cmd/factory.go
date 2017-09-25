package cmd

import (
	"fmt"
	"net/http"

	davclient "github.com/cloudfoundry/bosh-davcli/client"
	davconf "github.com/cloudfoundry/bosh-davcli/config"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type Factory interface {
	Create(name string) (cmd Cmd, err error)
	SetConfig(config davconf.Config)
}

func NewFactory(logger boshlog.Logger) Factory {
	return &factory{
		cmds:   make(map[string]Cmd),
		logger: logger,
	}
}

type factory struct {
	config davconf.Config
	cmds   map[string]Cmd
	logger boshlog.Logger
}

func (f *factory) Create(name string) (cmd Cmd, err error) {
	cmd, found := f.cmds[name]
	if !found {
		err = fmt.Errorf("Could not find command with name %s", name)
	}
	return
}

func (f *factory) SetConfig(config davconf.Config) {
	client := davclient.NewClient(config, http.DefaultClient, f.logger)

	f.cmds = map[string]Cmd{
		"put": newPutCmd(client),
		"get": newGetCmd(client),
	}
}
