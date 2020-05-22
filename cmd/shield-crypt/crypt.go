package shieldcrypt

import (
	"crypto/cipher"
	"encoding/hex"
	"io"
	"os"
	"strings"

	"github.com/shieldproject/shield/core/vault"
)

const Encrypt = "encrypt"
const Decrypt = "decrypt"

func RunCrypt(mode string, in io.Reader, out io.Writer, key string, IV string, enctype string) {
	var encStream, decStream cipher.Stream

	keyRaw, err := hex.DecodeString(strings.Replace(key, "-", "", -1))
	if err != nil {
		panic(err)
	}
	ivRaw, err := hex.DecodeString(strings.Replace(IV, "-", "", -1))
	if err != nil {
		panic(err)
	}

	if enctype != "" {
		encStream, decStream, err = vault.Stream(enctype, []byte(keyRaw), []byte(ivRaw))
		if err != nil {
			panic(err)
		}
	} else {
		if _, err := io.Copy(out, in); err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	if mode == Encrypt {
		encrypter := cipher.StreamWriter{
			S: encStream,
			W: out,
		}
		if _, err := io.Copy(encrypter, in); err != nil {
			panic(err)
		}
	}

	if mode == Decrypt {
		decrypter := cipher.StreamReader{
			S: decStream,
			R: in,
		}
		if _, err := io.Copy(out, decrypter); err != nil {
			panic(err)
		}
	}
	os.Exit(0)
}
