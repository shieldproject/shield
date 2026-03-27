package main

import (
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/core/vault"
)

var Version = ""

func main() {
	var encrypt bool
	var decrypt bool

	rootCmd := &cobra.Command{
		Use:   "shield-crypt",
		Short: "Pipeline worker for encrypting / decrypting",
		Long: `shield-crypt - Pipeline worker (shield-pipe) for encrypting / decrypting

Key material is read in as a JSON object, from file descriptor 3.
The following keys must be set:

  enc_key    - Secret key, hex-encoded.
  enc_iv     - Initialization vector, hex-encoded.
  enc_type   - The cipher and chaining mode to use.

Note: you probably don't want to run this yourself, unless
you know *exactly* what you are doing.`,
		Version: func() string {
			if Version == "" || Version == "dev" {
				return "development"
			}
			return Version
		}(),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var encStream, decStream cipher.Stream
			var crypt struct {
				Key  string `json:"enc_key"`
				IV   string `json:"enc_iv"`
				Type string `json:"enc_type"`
			}

			if encrypt && decrypt {
				fmt.Fprintf(os.Stderr, "Both encrypting and decrypting flags were set.\n")
				fmt.Fprintf(os.Stderr, "Cowardly refusing to run.\n")
				os.Exit(1)
			}

			decoder := json.NewDecoder(os.NewFile(uintptr(3), "fd3"))
			if err := decoder.Decode(&crypt); err == nil {
				keyRaw, err := hex.DecodeString(strings.Replace(crypt.Key, "-", "", -1))
				if err != nil {
					panic(err)
				}
				ivRaw, err := hex.DecodeString(strings.Replace(crypt.IV, "-", "", -1))
				if err != nil {
					panic(err)
				}

				if crypt.Type != "" {
					encStream, decStream, err = vault.Stream(crypt.Type, []byte(keyRaw), []byte(ivRaw))
					if err != nil {
						panic(err)
					}
				} else {
					if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
						panic(err)
					}
					os.Exit(0)
				}
			}

			if encrypt {
				encrypter := cipher.StreamWriter{
					S: encStream,
					W: os.Stdout,
				}
				if _, err := io.Copy(encrypter, os.Stdin); err != nil {
					panic(err)
				}
			}

			if decrypt {
				decrypter := cipher.StreamReader{
					S: decStream,
					R: os.Stdin,
				}
				if _, err := io.Copy(os.Stdout, decrypter); err != nil {
					panic(err)
				}
			}
			return nil
		},
	}

	rootCmd.Flags().BoolVarP(&encrypt, "encrypt", "e", false, "Perform encryption of the plaintext on stdin -> stdout")
	rootCmd.Flags().BoolVarP(&decrypt, "decrypt", "d", false, "Perform decryption of the ciphertext on stdin -> stdout")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
