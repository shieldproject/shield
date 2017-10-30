package backends

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"strings"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//DisplayCurrent displays information about the currently targeted backend to
//the screen
func DisplayCurrent() {
	cur := config.Current()
	if cur == nil {
		fmt.Fprintf(os.Stderr, "No current SHIELD backend\n\n")
	} else {
		fmt.Fprintf(os.Stderr, "Using @G{%s} (%s) as SHIELD backend\n\n", cur.Address, cur.Name)
	}
}

//DisplayCACert displays the CA Cert for the currently targeted backend
func DisplayCACert() {
	cur := config.Current()
	if cur == nil {
		fmt.Fprintf(os.Stderr, "No current SHIELD backend\n\n")
	} else if cur.CACert == "" {
		fmt.Fprintf(os.Stderr, "Current backend {%s} does not have a CA Cert configured", cur.Name)
	} else {
		fmt.Fprintf(os.Stderr, cur.CACert)
	}
}

//ParseCACertFlag inteprets the input as a PEM encoded cert block if it has a
// PEM header, and tries to use it as a filename otherwise. Returns the PEM block
// if successful.
func ParseCACertFlag(input string) (cert string, err error) {
	if looksLikePEMCert(input) {
		log.DEBUG("Interpreting ca cert flag as PEM cert")
		cert, err = certificatePEMBlock(input)
	} else {
		log.DEBUG("Interpreting ca cert flag as filename")
		cert, err = certificatePEMBlockFromFile(input)
	}

	return
}

func certificatePEMBlockFromFile(filepath string) (string, error) {
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("Could not read CA Cert file `%s': %s", filepath, err.Error())
	}

	return certificatePEMBlock(string(fileBytes))
}

func certificatePEMBlock(input string) (string, error) {
	block, rest := pem.Decode([]byte(input))
	if block == nil {
		return "", fmt.Errorf("Failed to decode PEM block")
	}
	if len(strings.TrimSpace(string(rest))) > 0 {
		return "", fmt.Errorf("Extra contents found in cert (is this a cert bundle?)")
	}
	if block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("PEM Block type is not CERTIFICATE")
	}

	//Check that this is a well-formatted cert
	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("PEM Block does not contain a valid certificate: %s", err.Error())
	}

	return string(pem.EncodeToMemory(block)), nil
}

func looksLikePEMCert(input string) bool {
	return strings.HasPrefix(strings.TrimSpace(input), "-----BEGIN CERTIFICATE-----")
}
