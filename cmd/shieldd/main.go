package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	log "github.com/shieldproject/shield/internal/log"

	// sql drivers
	_ "github.com/mattn/go-sqlite3"

	"github.com/shieldproject/shield/core"
)

var Version = ""

var (
	configFile string
	logLevel   string
)

var rootCmd = &cobra.Command{
	Use:           "shieldd",
	Short:         "The SHIELD Core daemon",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("config") {
			if v := os.Getenv("SHIELD_CONFIG_FILE"); v != "" {
				configFile = v
			}
		}
		if !cmd.Flags().Changed("log-level") {
			if v := os.Getenv("SHIELD_LOG_LEVEL"); v != "" {
				logLevel = v
			}
		}

		log.SetupLogging(logLevel)
		log.Infof("starting up shield core")

		c, err := core.Configure(configFile, core.DefaultConfig)
		if err != nil {
			log.Errorf("shield core failed to start up: %s", err)
			os.Exit(1)
		}

		c.Main()
		return nil
	},
}

func init() {
	core.Version = Version
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to the SHIELD Core configuration file")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "What messages to log (error, warning, info, or debug)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "!!! %s\n", err)
		os.Exit(1)
	}
}
