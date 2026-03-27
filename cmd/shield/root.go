package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	optQuiet  bool
	optYes    bool
	optDebug  bool
	optTrace  bool
	optBatch  bool
	optCore   string
	optConfig string
	optJSON   bool
	optLong   bool
	optExact  bool
	optFuzzy  bool
	optTenant string
)

var cliConfig *Config

var rootCmd = &cobra.Command{
	Use:           "shield",
	Short:         "SHIELD backup management CLI",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Read env vars for flags not explicitly set
		if !cmd.Flags().Changed("debug") {
			if v := os.Getenv("SHIELD_DEBUG"); v != "" {
				optDebug = true
			}
		}
		if !cmd.Flags().Changed("trace") {
			if v := os.Getenv("SHIELD_TRACE"); v != "" {
				optTrace = true
			}
		}
		if !cmd.Flags().Changed("batch") {
			if v := os.Getenv("SHIELD_BATCH_MODE"); v != "" {
				optBatch = true
			}
		}
		if !cmd.Flags().Changed("core") {
			if v := os.Getenv("SHIELD_CORE"); v != "" {
				optCore = v
			}
		}
		if !cmd.Flags().Changed("config") {
			if v := os.Getenv("SHIELD_CLI_CONFIG"); v != "" {
				optConfig = v
			}
		}
		if !cmd.Flags().Changed("json") {
			if v := os.Getenv("SHIELD_JSON_MODE"); v != "" {
				optJSON = true
			}
		}
		if !cmd.Flags().Changed("tenant") {
			if v := os.Getenv("SHIELD_TENANT"); v != "" {
				optTenant = v
			}
		}

		// Flag interdependencies
		if optJSON {
			optYes = true
		}
		if optBatch {
			optYes = true
		}
		if optQuiet {
			optDebug = false
			optTrace = false
		}
		if optFuzzy {
			optExact = false
		}

		// Load config
		var err error
		cliConfig, err = ReadConfig(optConfig, optConfig+"_config")
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	home, _ := os.UserHomeDir()
	defaultConfig := home + "/.shield"

	rootCmd.Version = Version

	f := rootCmd.PersistentFlags()
	f.BoolVarP(&optQuiet, "quiet", "q", false, "Suppress non-essential output")
	f.BoolVarP(&optYes, "yes", "y", false, "Answer yes to all prompts")
	f.BoolVarP(&optDebug, "debug", "D", false, "Enable debug output")
	f.BoolVarP(&optTrace, "trace", "T", false, "Enable trace output")
	f.BoolVarP(&optBatch, "batch", "b", false, "Enable batch (non-interactive) mode")
	f.StringVarP(&optCore, "core", "c", "", "SHIELD core to connect to")
	f.StringVar(&optConfig, "config", defaultConfig, "Path to SHIELD config file")
	f.BoolVar(&optJSON, "json", false, "Output in JSON format")
	f.BoolVarP(&optLong, "long", "L", false, "Show extended output")
	f.BoolVar(&optExact, "exact", false, "Exact matching for searches")
	f.BoolVar(&optFuzzy, "fuzzy", false, "Fuzzy matching for searches")
	f.StringVarP(&optTenant, "tenant", "t", "", "SHIELD tenant to operate in")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
