package main

import (
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/jhunt/go-cli"
	"github.com/starkandwayne/shield/crypter"
)

var opt struct {
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
			encStream, decStream, err = crypter.Stream(crypt.Type, []byte(keyRaw), []byte(ivRaw))
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

	_, _, err := cli.Parse(&opt)

	if err != nil {
		panic(err)
	}

	if opt.Encrypt && opt.Decrypt {
		os.Stderr.WriteString("Both encrypting and decrypting flags were set. Cowardly refusing to run.")
		os.Exit(1)
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
