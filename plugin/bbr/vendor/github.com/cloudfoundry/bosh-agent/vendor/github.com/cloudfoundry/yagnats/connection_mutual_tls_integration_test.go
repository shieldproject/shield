package yagnats

import (
	"crypto/x509"
	"os/exec"

	"errors"
	. "gopkg.in/check.v1"
)

type MutualTLSSuite struct {
	Client   *Client
	NatsConn NATSConn
	NatsCmd  *exec.Cmd
}

var _ = Suite(&MutualTLSSuite{})

func (t *MutualTLSSuite) SetUpSuite(c *C) {
	t.NatsCmd = startNatsMutualTLS(4556)
}

func (t *MutualTLSSuite) TearDownSuite(c *C) {
	stopCmd(t.NatsCmd)
}

func (t *MutualTLSSuite) TestNewMutualTLSConnection(c *C) {
	client := NewClient()

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(ValidCA)
	c.Assert(ok, Equals, true)

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4556",
		Username:   "nats",
		Password:   "nats",
		CertPool:   roots,
		ClientCert: &ValidClientCert,
	})
	c.Assert(err, IsNil)
	t.Client = client

	pingSuccess := client.Ping()
	c.Assert(pingSuccess, Equals, true)
}

func (t *MutualTLSSuite) TestNewMutualTLSConnectionWithWrongCA(c *C) {
	client := NewClient()

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(InvalidCA)
	c.Assert(ok, Equals, true)

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4556",
		Username:   "nats",
		Password:   "nats",
		CertPool:   roots,
		ClientCert: &ValidClientCert,
	})

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "^x509: certificate signed by unknown authority.*$")
}

func (t *MutualTLSSuite) TestNewMutualTLSConnectionWithEmptyCertPool(c *C) {
	client := NewClient()

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4556",
		Username:   "nats",
		Password:   "nats",
		CertPool:   x509.NewCertPool(),
		ClientCert: &ValidClientCert,
	})

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "x509: certificate signed by unknown authority")
}

func (t *MutualTLSSuite) TestNewMutualTLSConnectionWithInvalidClientCert(c *C) {
	client := NewClient()

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(ValidCA)
	c.Assert(ok, Equals, true)

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4556",
		Username:   "nats",
		Password:   "nats",
		CertPool:   roots,
		ClientCert: &InvalidClientCert,
	})

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "remote error: tls: bad certificate")

	pingSuccess := client.Ping()
	c.Assert(pingSuccess, Equals, false)
}

func (t *MutualTLSSuite) TestNewMutualTLSConnectionWithNoClientCert(c *C) {
	client := NewClient()

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(ValidCA)
	c.Assert(ok, Equals, true)

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4556",
		Username: "nats",
		Password: "nats",
		CertPool: roots,
	})

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "remote error: tls: bad certificate")

	pingSuccess := client.Ping()
	c.Assert(pingSuccess, Equals, false)
}

func (t *MutualTLSSuite) TestNewMutualTLSConnectionWithVerifyPeerCertificateCallbackSuccess(c *C) {
	client := NewClient()

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(ValidCA)
	c.Assert(ok, Equals, true)

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4556",
		Username:   "nats",
		Password:   "nats",
		CertPool:   roots,
		ClientCert: &ValidClientCert,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			for _, chain := range verifiedChains {
				org := chain[len(chain)-1].Subject.Organization
				if len(org) >= 1 && org[0] == "Cloud Foundry" {
					return nil
				}
			}
			return errors.New("Unexpected organization.")
		},
	})
	c.Assert(err, IsNil)
	t.Client = client

	pingSuccess := client.Ping()
	c.Assert(pingSuccess, Equals, true)
}

func (t *MutualTLSSuite) TestNewMutualTLSConnectionWithVerifyPeerCertificateCallbackError(c *C) {
	client := NewClient()

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(ValidCA)
	c.Assert(ok, Equals, true)

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4556",
		Username:   "nats",
		Password:   "nats",
		CertPool:   roots,
		ClientCert: &ValidClientCert,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			for _, chain := range verifiedChains {
				org := chain[len(chain)-1].Subject.Organization
				if len(org) >= 1 && org[0] == "Yagnats" {
					return nil
				}
			}
			return errors.New("Unexpected organization.")
		},
	})
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Unexpected organization.")

	t.Client = client

	pingSuccess := client.Ping()
	c.Assert(pingSuccess, Equals, false)
}
