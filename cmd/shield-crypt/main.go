package main

import (
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jhunt/go-cli"

	"github.com/starkandwayne/shield/core/vault"
)

var Version = ""

var opt struct {
	Help    bool `cli:"-h, --help"`
	Version bool `cli:"-v, --version"`

	Encrypt bool `cli:"-e, --encrypt"`
	Decrypt bool `cli:"-d, --decrypt"`
}

func main() {
	var encStream, decStream cipher.Stream
	var crypt struct {
		Key  string `json:"enc_key"`
		IV   string `json:"enc_iv"`
		Type string `json:"enc_type"`
	}

	_, args, err := cli.Parse(&opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! %s\n", err)
		os.Exit(1)
	}
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "!!! extra arguments found\n")
		os.Exit(1)
	}

	if opt.Help {
		fmt.Fprintf(os.Stderr, "shield-crypt - Pipeline worker (shield-pipe) for encrypting / decrypting\n\n")
		fmt.Fprintf(os.Stderr, "Options\n")
		fmt.Fprintf(os.Stderr, "  -h, --help       Show this help screen.\n")
		fmt.Fprintf(os.Stderr, "  -v, --version    Display the SHIELD version.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  -e, --encrypt    Perform encryption of the plaintext on stdin -> stdout.\n")
		fmt.Fprintf(os.Stderr, "  -d, --decrypt    Perform decryption of the ciphertext on stdin -> stdout.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Key material is read in as a JSON object, from file descriptor 3.\n")
		fmt.Fprintf(os.Stderr, "The following keys myst be set:\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  enc_key    - Secret key, hex-encoded.\n")
		fmt.Fprintf(os.Stderr, "  enc_iv     - Initiaization vector, hex-encoded.\n")
		fmt.Fprintf(os.Stderr, "  enc_type   - The cipher and chaining mode to use.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Note: you probably don't want to run this yourself, unless\n")
		fmt.Fprintf(os.Stderr, "you know *exactly* what you are doing.\n")
		os.Exit(0)
	}

	if opt.Version {
		if Version == "" || Version == "dev" {
			fmt.Fprintf(os.Stderr, "shield-crypt (development)\n")
		} else {
			fmt.Fprintf(os.Stderr, "shield-crypt v%s\n", Version)
		}
		os.Exit(0)
	}

	if opt.Encrypt && opt.Decrypt {
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

	if opt.Encrypt {
		encrypter := cipher.StreamWriter{
			S: encStream,
			W: os.Stdout,
		}
		if _, err := io.Copy(encrypter, os.Stdin); err != nil {
			panic(err)
		}
	}

	if opt.Decrypt {
		decrypter := cipher.StreamReader{
			S: decStream,
			R: os.Stdin,
		}
		if _, err := io.Copy(os.Stdout, decrypter); err != nil {
			panic(err)
		}
	}
	os.Exit(0)
}
