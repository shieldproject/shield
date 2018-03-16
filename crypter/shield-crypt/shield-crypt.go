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

type CLIOpts struct {
	EncryptShort bool `cli:"-e"`
	DecryptShort bool `cli:"-d"`
	EncryptFull  bool `cli:"--encrypt"`
	DecryptFull  bool `cli:"--decrypt"`
}

func main() {
	var encStream, decStream cipher.Stream
	var data struct {
		EncryptionKey  string `json:"enc_key"`
		EncryptionIV   string `json:"enc_iv"`
		EncryptionType string `json:"enc_type"`
	}

	decoder := json.NewDecoder(os.NewFile(uintptr(3), "encConfig"))
	if err := decoder.Decode(&data); err == nil {
		keyRaw, err := hex.DecodeString(strings.Replace(data.EncryptionKey, "-", "", -1))
		if err != nil {
			panic(err)
		}
		ivRaw, err := hex.DecodeString(strings.Replace(data.EncryptionIV, "-", "", -1))
		if err != nil {
			panic(err)
		}

		if data.EncryptionType != "" {
			encStream, decStream, err = crypter.Stream(data.EncryptionType, []byte(keyRaw), []byte(ivRaw))
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

	var opts CLIOpts

	_, _, err := cli.Parse(&opts)

	if err != nil {
		panic(err)
	}

	if opts.EncryptShort || opts.EncryptFull {
		encrypter := cipher.StreamWriter{
			S: encStream,
			W: os.Stdout,
		}
		if _, err := io.Copy(encrypter, os.Stdin); err != nil {
			panic(err)
		}
	}

	if opts.DecryptShort || opts.DecryptFull {
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
