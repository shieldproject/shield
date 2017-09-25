package app

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	davcmd "github.com/cloudfoundry/bosh-davcli/cmd"
	davconfig "github.com/cloudfoundry/bosh-davcli/config"
	"io/ioutil"
	"os"
)

type App struct {
	runner davcmd.Runner
}

func New(runner davcmd.Runner) (app App) {
	app.runner = runner
	return
}

func (app App) Run(args []string) (err error) {
	args = args[1:]
	var configFilePath string
	var printVersion bool

	flagSet := flag.NewFlagSet("davcli-args", flag.ContinueOnError)
	flagSet.StringVar(&configFilePath, "c", "", "Config file path")
	flagSet.BoolVar(&printVersion, "v", false, "print version info")

	err = flagSet.Parse(args)
	if err != nil {
		return
	}

	if printVersion {
		fmt.Println("davcli version [[version]]")
		return
	}

	if configFilePath == "" {
		err = errors.New("Config file arg `-c` is missing")
		return
	}

	file, err := os.Open(configFilePath)
	if err != nil {
		return
	}

	configBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	config := davconfig.Config{}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return
	}

	app.runner.SetConfig(config)
	err = app.runner.Run(args[2:])
	return
}
