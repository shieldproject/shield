package yagnats

import (
	"crypto/tls"
	"io/ioutil"
)

func init() {
	ValidCA, _ = ioutil.ReadFile("./assets/ca.pem")
	InvalidCA, _ = ioutil.ReadFile("./assets/invalid-ca.pem")
	ValidClientCert, _ = tls.LoadX509KeyPair("./assets/client-cert.pem", "./assets/client-pkey.pem")
	InvalidClientCert, _ = tls.LoadX509KeyPair("./assets/client-invalid-cert.pem", "./assets/client-invalid-pkey.pem")
}

var ValidCA []byte
var InvalidCA []byte
var ValidClientCert tls.Certificate
var InvalidClientCert tls.Certificate
