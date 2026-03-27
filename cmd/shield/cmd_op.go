package main

import (
	"io/ioutil"
	"os"
	"regexp"

	fmt "github.com/jhunt/go-ansi"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/core/vault"
)

var opCmd = &cobra.Command{
	Use:   "op",
	Short: "Operator low-level commands",
}

var opPryCmd = &cobra.Command{
	Use:   "pry /path/to/vault.crypt",
	Short: "Decrypt a SHIELD vault crypt file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		master := secureprompt("@Y{SHIELD Master Password:} ")
		creds, err := vault.ReadCrypt(args[0], master)
		bail(err)
		fmt.Printf("@C{Seal Key:}   %s\n", creds.SealKey)
		fmt.Printf("@C{Root Token:} %s\n", creds.RootToken)
		return nil
	},
}

var opIfkCmd = &cobra.Command{
	Use:   "ifk",
	Short: "Derive fixed-key encryption parameters from stdin",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := ioutil.ReadAll(os.Stdin)
		bail(err)

		key := regexp.MustCompile(`\s`).ReplaceAll(b, nil)
		enc, err := vault.DeriveFixedParameters(key)
		bail(err)

		fmt.Printf("@C{Cipher:} %s\n", enc.Type)
		fmt.Printf("@C{Key:}    %s\n", enc.Key)
		fmt.Printf("@C{IV:}     %s\n", enc.IV)

		if enc.Type == "aes256-ctr" {
			fmt.Printf("\n@G{OpenSSL} decryption command:\n")
			fmt.Printf("  openssl enc -d -md sha256 -aes-256-ctr -K %s -iv %s < file\n\n", enc.Key, enc.IV)
		}
		return nil
	},
}

func init() {
	opCmd.AddCommand(opPryCmd)
	opCmd.AddCommand(opIfkCmd)
	rootCmd.AddCommand(opCmd)
}
