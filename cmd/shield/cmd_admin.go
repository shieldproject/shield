package main

import (
	"os"

	fmt "github.com/jhunt/go-ansi"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/client/v2/shield"
)

var (
	adminInitMaster   string
	adminUnlockMaster string
	adminRekeyOld     string
	adminRekeyNew     string
	adminRekeyRotate  bool
)

var initCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"initialize"},
	Short:   "Initialize a new SHIELD Core",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		master := adminInitMaster
		if master == "" {
			master = os.Getenv("SHIELD_CORE_MASTER")
		}
		if master == "" {
			a := secureprompt("@Y{New SHIELD Core master password}: ")
			b := secureprompt("@Y{Confirm new master password}: ")
			if a == "" {
				fail(3, "@R{master password cannot be blank!}\n")
			} else if a != b {
				fail(3, "@R{master password mismatch!}\n")
			}
			master = a
		}
		fixedKey, err := c.Initialize(master)
		bail(err)

		fmt.Printf("SHIELD core unlocked successfully.\n")

		if fixedKey != "" {
			fmt.Printf("@R{BELOW IS YOUR FIXED KEY FOR RECOVERING FIXED-KEY BACKUPS.}\n")
			fmt.Printf("@R{SAVE THIS IN A SECURE LOCATION.}\n")
			fmt.Printf("----------------------------------------------------------------\n")
			fmt.Printf("@Y{" + c.SplitKey(fixedKey, 64) + "}")
			fmt.Printf("\n----------------------------------------------------------------\n")
		} else {
			bail(fmt.Errorf("Failed to initialize Fixed Key!"))
		}
		return nil
	},
}

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock a SHIELD Core",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		bail(c.Lock())
		fmt.Printf("SHIELD core locked successfully.\n")
		return nil
	},
}

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock a SHIELD Core",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		master := adminUnlockMaster
		if master == "" {
			master = os.Getenv("SHIELD_CORE_MASTER")
		}
		if master == "" {
			master = secureprompt("@Y{SHIELD Core master password:} ")
		}
		err := c.Unlock(master)
		bail(err)
		fmt.Printf("SHIELD core unlocked successfully.\n")
		return nil
	},
}

var rekeyCmd = &cobra.Command{
	Use:   "rekey",
	Short: "Change a SHIELD Core master password",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		oldMaster := adminRekeyOld
		newMaster := adminRekeyNew

		if oldMaster == "" {
			oldMaster = os.Getenv("SHIELD_CORE_MASTER")
		}
		if oldMaster == "" {
			oldMaster = secureprompt("@Y{Current master password:} ")
		}
		if newMaster == "" {
			a := secureprompt("@C{New SHIELD Core master password:} ")
			b := secureprompt("@C{Confirm new master password}: ")
			if a == "" {
				fail(3, "@R{master password cannot be blank!}\n")
			} else if a != b {
				fail(3, "@R{new master password mismatch!}\n")
			}
			newMaster = a
		}
		fixedKey, err := c.Rekey(oldMaster, newMaster, adminRekeyRotate)
		bail(err)

		if fixedKey != "" {
			fmt.Printf("@R{BELOW IS YOUR FIXED KEY FOR RECOVERING FIXED-KEY BACKUPS.}\n")
			fmt.Printf("@R{SAVE THIS IN A SECURE LOCATION.}\n")
			fmt.Printf("----------------------------------------------------------------\n")
			fmt.Printf("@Y{" + c.SplitKey(fixedKey, 64) + "}")
			fmt.Printf("\n----------------------------------------------------------------\n")
		} else if adminRekeyRotate {
			bail(fmt.Errorf("Failed to initialize Fixed Key!"))
		}

		fmt.Printf("SHIELD core rekeyed successfully.\n")
		return nil
	},
}

// clientFromConfig builds a shield.Client from the current cliConfig selection.
// Commands that need a client call this after PersistentPreRunE has loaded cliConfig.
func clientFromConfig() *shield.Client {
	bail(cliConfig.Select(optCore))
	return &shield.Client{
		URL:                cliConfig.Current.URL,
		Debug:              optDebug,
		Trace:              optTrace,
		Session:            cliConfig.Current.Session,
		InsecureSkipVerify: cliConfig.Current.InsecureSkipVerify,
		CACertificate:      cliConfig.Current.CACertificate,
		TrustSystemCAs:     true,
	}
}

func init() {
	initCmd.Flags().StringVar(&adminInitMaster, "master", "", "Master password for initialization")
	unlockCmd.Flags().StringVar(&adminUnlockMaster, "master", "", "Master password to unlock with")
	rekeyCmd.Flags().StringVar(&adminRekeyOld, "old-master", "", "Current master password")
	rekeyCmd.Flags().StringVar(&adminRekeyNew, "new-master", "", "New master password")
	rekeyCmd.Flags().BoolVar(&adminRekeyRotate, "rotate-fixed-key", false, "Rotate fixed key as well")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(lockCmd)
	rootCmd.AddCommand(unlockCmd)
	rootCmd.AddCommand(rekeyCmd)
}
