package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var Version = ""

func main() {
	var compression string

	rootCmd := &cobra.Command{
		Use:   "shield-report",
		Short: "Pipeline worker for reporting",
		Long:  "shield-report - Pipeline worker (shield-pipe) for reporting",
		Version: func() string {
			if Version == "" || Version == "dev" {
				return "development"
			}
			return Version
		}(),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "!!! shield-report utility failed to read standard input: %s\n", err)
				os.Exit(3)
			}

			var data map[string]interface{}
			if err := json.Unmarshal(b, &data); err != nil {
				fmt.Fprintf(os.Stderr, "!!! shield-report utility failed to parse JSON from standard input: %s\n", err)
				os.Exit(3)
			}

			if compression != "" {
				data["compression"] = compression
			}

			b, err = json.Marshal(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "!!! shield-report utility failed to encode output JSON: %s\n", err)
				os.Exit(4)
			}
			fmt.Printf("%s\n", string(b))
			return nil
		},
	}

	rootCmd.Flags().StringVarP(&compression, "compression", "c", "", `Set the "compression" key in the output JSON`)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
