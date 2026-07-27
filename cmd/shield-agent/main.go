package main

import (
	"fmt"
	"os"

	log "github.com/shieldproject/shield/internal/log"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/agent"
)

var Version = ""

var (
	configFile string
	logLevel   string
)

var rootCmd = &cobra.Command{
	Use:           "shield-agent",
	Short:         "Run a remote SHIELD orchestration agent",
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("config") {
			if v := os.Getenv("SHIELD_AGENT_CONFIG_FILE"); v != "" {
				configFile = v
			}
		}
		if !cmd.Flags().Changed("log-level") {
			if v := os.Getenv("SHIELD_AGENT_LOG_LEVEL"); v != "" {
				logLevel = v
			}
		}

		log.SetupLogging(logLevel)
		log.Infof("starting agent")

		ag := agent.NewAgent()
		ag.Version = Version
		if err := ag.ReadConfig(configFile); err != nil {
			log.Errorf("configuration failed: %s", err)
			return err
		}
		ag.Run()
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to the agent configuration file")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "What messages to log (error, warning, info, or debug)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "!!! %s\n", err)
		os.Exit(1)
	}
}
