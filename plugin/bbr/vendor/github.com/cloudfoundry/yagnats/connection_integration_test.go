package yagnats

import (
	"crypto/x509"
	"os/exec"

	. "gopkg.in/check.v1"
)

type TLSSuite struct {
	Client   *Client
	NatsConn NATSConn
	NatsCmd  *exec.Cmd
}

var _ = Suite(&TLSSuite{})

func (t *TLSSuite) SetUpSuite(c *C) {
	t.NatsCmd = startNatsTLS(4555)
	waitUntilNatsUp(4555)
}

func (t *TLSSuite) TearDownSuite(c *C) {
	stopCmd(t.NatsCmd)
}

func (t *TLSSuite) TestNewTLSConnection(c *C) {
	client := NewClient()

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(ValidCA)
	c.Assert(ok, Equals, true)

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4555",
		Username: "nats",
		Password: "nats",
		CertPool: roots,
	})
	c.Assert(err, IsNil)
	t.Client = client

	pingSuccess := client.Ping()
	c.Assert(pingSuccess, Equals, true)
}

func (t *TLSSuite) TestNewTLSConnectionWithWrongCA(c *C) {
	client := NewClient()

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(InvalidCA)
	c.Assert(ok, Equals, true)

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4555",
		Username: "nats",
		Password: "nats",
		CertPool: roots,
	})

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "^x509: certificate signed by unknown authority.*$")

}

func (t *TLSSuite) TestNewTLSConnectionWithEmptyCertPool(c *C) {
	client := NewClient()

	err := client.Connect(&ConnectionInfo{Addr: "127.0.0.1:4555",
		Username: "nats",
		Password: "nats",
		CertPool: x509.NewCertPool(),
	})

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "x509: certificate signed by unknown authority")

}
