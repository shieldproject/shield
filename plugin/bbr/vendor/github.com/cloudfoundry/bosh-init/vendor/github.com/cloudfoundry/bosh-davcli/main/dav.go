package main

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/bosh-davcli/app"
	"github.com/cloudfoundry/bosh-davcli/cmd"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func main() {
	logger := boshlog.NewLogger(boshlog.LevelNone)
	cmdFactory := cmd.NewFactory(logger)

	cmdRunner := cmd.NewRunner(cmdFactory)

	cli := app.New(cmdRunner)

	err := cli.Run(os.Args)
	if err != nil {
		fmt.Printf("Error running app - %s", err.Error())
		os.Exit(1)
	}
}
