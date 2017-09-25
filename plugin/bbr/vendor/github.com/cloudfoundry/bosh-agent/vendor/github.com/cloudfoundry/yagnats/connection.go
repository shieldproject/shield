package yagnats

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"sync"
	"time"
)

type Connection struct {
	conn net.Conn

	addr string
	user string
	pass string

	dial func(network, address string) (net.Conn, error)

	writeLock *sync.Mutex

	pongs chan *PongPacket
	oks   chan *OKPacket
	errs  chan error

	onMessage func(*MsgPacket)

	Disconnected chan bool

	logger      Logger
	loggerMutex *sync.RWMutex
}

type ConnectionProvider interface {
	ProvideConnection() (*Connection, error)
}

func NewConnection(addr, user, pass string) *Connection {
	return &Connection{
		addr: addr,
		user: user,
		pass: pass,

		dial: func(network, address string) (net.Conn, error) {
			return net.DialTimeout(network, address, 5*time.Second)
		},

		writeLock: &sync.Mutex{},

		logger:      &DefaultLogger{},
		loggerMutex: &sync.RWMutex{},

		pongs: make(chan *PongPacket),

		oks: make(chan *OKPacket),

		// buffer size of 1 to account for fatal unexpected errors
		// from the server (i.e. slow consumer)
		errs: make(chan error, 1),

		Disconnected: make(chan bool),
	}
}

func NewTLSConnection(addr, user, pass string, certPool *x509.CertPool, clientCert *tls.Certificate, verifyPeerCertificate func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error) *Connection {
	connection := NewConnection(addr, user, pass)
	connection.dial = func(network, address string) (net.Conn, error) {
		conn, err := net.DialTimeout(network, address, 5*time.Second)
		if err != nil {
			return nil, err
		}

		br := bufio.NewReaderSize(conn, 32768)
		_, err = Parse(br)
		if err != nil {
			return nil, err
		}

		hostname, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		config := tls.Config{
			RootCAs:               certPool,
			ServerName:            hostname,
			VerifyPeerCertificate: verifyPeerCertificate,
		}

		// When client certificate is provided, we are expecting mutual TLS.
		if clientCert != nil {
			config.Certificates = []tls.Certificate{*clientCert}
		}

		conn = tls.Client(conn, &config)

		tlsConn := conn.(*tls.Conn)
		err = tlsConn.Handshake()
		return tlsConn, err
	}
	return connection
}

type ConnectionInfo struct {
	Addr     string
	Username string
	Password string
	Dial     func(network, address string) (net.Conn, error)

	CertPool              *x509.CertPool
	ClientCert            *tls.Certificate
	VerifyPeerCertificate func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error
}

func (c *ConnectionInfo) ProvideConnection() (*Connection, error) {
	var conn *Connection
	if c.CertPool == nil {
		conn = NewConnection(c.Addr, c.Username, c.Password)
	} else {
		conn = NewTLSConnection(c.Addr, c.Username, c.Password, c.CertPool, c.ClientCert, c.VerifyPeerCertificate)
	}

	if c.Dial != nil {
		conn.dial = c.Dial
	}

	var err error

	err = conn.Dial()
	if err != nil {
		return nil, err
	}

	err = conn.Handshake()
	if err != nil {
		return nil, err
	}

	return conn, nil
}

type ConnectionCluster struct {
	Members []ConnectionProvider
}

func (c *ConnectionCluster) ProvideConnection() (conn *Connection, err error) {
	for _, cp := range c.Members {
		conn, err = cp.ProvideConnection()
		if err == nil {
			return conn, nil
		}
	}
	return nil, err
}

func (c *Connection) Dial() error {
	conn, err := c.dial("tcp", c.addr)
	if err != nil {
		return err
	}

	c.conn = conn

	go c.receivePackets()

	return nil
}

func (c *Connection) OnMessage(callback func(*MsgPacket)) {
	c.onMessage = callback
}

func (c *Connection) Handshake() error {
	c.Send(&ConnectPacket{User: c.user, Pass: c.pass})
	return c.ErrOrOK()
}

func (c *Connection) Disconnect() {
	c.conn.Close()
}

func (c *Connection) ErrOrOK() error {
	c.Logger().Debug("connection.err-or-ok.wait")
	select {
	case err := <-c.errs:
		c.Logger().Warnd(map[string]interface{}{"error": err.Error()}, "connection.err-or-ok.err")
		return err
	case <-c.oks:
		c.Logger().Debug("connection.err-or-ok.ok")
		return nil
	}
}

func (c *Connection) Send(packet Packet) {
	c.Logger().Debugd(map[string]interface{}{"packet": packet}, "connection.packet.send")

	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	// ignore write errors; readPackets will notice connection being interrupted
	_, err := c.conn.Write(packet.Encode())
	if err != nil {
		c.Logger().Errord(map[string]interface{}{"error": err.Error()}, "connection.packet.write-error")
	}

	return
}

func (c *Connection) Ping() bool {
	c.Send(&PingPacket{})

	select {
	case _, ok := <-c.pongs:
		return ok
	case <-time.After(500 * time.Millisecond):
		return false
	}
}

func (c *Connection) SetLogger(logger Logger) {
	c.loggerMutex.Lock()
	c.logger = logger
	c.loggerMutex.Unlock()
}

func (c *Connection) Logger() Logger {
	c.loggerMutex.RLock()
	defer c.loggerMutex.RUnlock()

	return c.logger
}

func (c *Connection) receivePackets() {
	io := bufio.NewReader(c.conn)

	for {
		c.Logger().Debug("connection.packet.read")

		packet, err := Parse(io)
		if err != nil {
			c.Logger().Errord(map[string]interface{}{"error": err.Error()}, "connection.packet.read-error")
			c.Disconnect()
			c.disconnected()
			break
		}

		switch packet.(type) {
		case *PongPacket:
			c.Logger().Debug("connection.packet.pong-received")

			select {
			case c.pongs <- packet.(*PongPacket):
				c.Logger().Debug("connection.packet.pong-served")
			default:
				c.Logger().Debug("connection.packet.pong-unhandled")
			}

		case *PingPacket:
			c.Logger().Debug("connection.packet.ping-received")
			c.Send(&PongPacket{})

		case *OKPacket:
			c.Logger().Debug("connection.packet.ok-received")
			c.oks <- packet.(*OKPacket)

		case *ERRPacket:
			c.Logger().Debug("connection.packet.err-received")
			c.errs <- errors.New(packet.(*ERRPacket).Message)

		case *InfoPacket:
			c.Logger().Debug("connection.packet.info-received")
			// noop

		case *MsgPacket:
			c.Logger().Debugd(
				map[string]interface{}{"packet": packet},
				"connection.packet.msg-received",
			)

			c.onMessage(packet.(*MsgPacket))
		}
	}
}

func (c *Connection) disconnected() {
	c.Disconnected <- true
	c.errs <- errors.New("disconnected")
}
