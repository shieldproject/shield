package yagnats

import (
	"bytes"
	"sync"
	"time"

	. "gopkg.in/check.v1"
)

type CSuite struct {
	Connection *Connection
}

var _ = Suite(&CSuite{})

func (s *CSuite) SetUpTest(c *C) {
	s.Connection = NewConnection("foo", "bar", "baz")
}

func (s *CSuite) TestConnectionPong(c *C) {
	conn := &fakeConn{
		ReadBuffer:  bytes.NewBuffer([]byte("PING\r\n")),
		WriteBuffer: bytes.NewBuffer([]byte{}),
		WriteChan:   make(chan []byte),
	}

	// fill in a fake connection
	s.Connection.conn = conn
	go s.Connection.receivePackets()

	time.Sleep(1 * time.Second)

	waitReceive(c, "PONG\r\n", conn.WriteChan, 500)
}

func (s *CSuite) TestConnectionUnexpectedError(c *C) {
	conn := &fakeConn{
		ReadBuffer:  bytes.NewBuffer([]byte("-ERR 'foo'\r\nPING\r\n")),
		WriteBuffer: bytes.NewBuffer([]byte{}),
		WriteChan:   make(chan []byte),
	}

	// fill in a fake connection
	s.Connection.conn = conn
	go s.Connection.receivePackets()

	time.Sleep(1 * time.Second)

	waitReceive(c, "PONG\r\n", conn.WriteChan, 500)
}

func (s *CSuite) TestConnectionUnexpectedPong(c *C) {
	conn := &fakeConn{
		ReadBuffer:  bytes.NewBuffer([]byte("PONG\r\nPING\r\n")),
		WriteBuffer: bytes.NewBuffer([]byte{}),
		WriteChan:   make(chan []byte),
	}

	// fill in a fake connection
	s.Connection.conn = conn
	go s.Connection.receivePackets()

	time.Sleep(1 * time.Second)

	waitReceive(c, "PONG\r\n", conn.WriteChan, 500)
}

func (s *CSuite) TestConnectionDisconnect(c *C) {
	conn := &fakeConn{
		ReadBuffer:  bytes.NewBuffer([]byte{}),
		WriteBuffer: bytes.NewBuffer([]byte{}),
		WriteChan:   make(chan []byte),
		Closed:      false,
	}

	// fill in a fake connection
	s.Connection.conn = conn
	c.Assert(conn.Closed, Equals, false)

	s.Connection.Disconnect()

	c.Assert(conn.Closed, Equals, true)
}

func (s *CSuite) TestConnectionDisconnectsOnParseErr(c *C) {
	conn := &fakeConn{
		ReadBuffer:  bytes.NewBuffer([]byte("!BAD\r\n")),
		WriteBuffer: bytes.NewBuffer([]byte{}),
		WriteChan:   make(chan []byte),
		Closed:      false,
	}

	// fill in a fake connection
	s.Connection.conn = conn
	go s.Connection.receivePackets()

	select {
	case <-s.Connection.Disconnected:
		c.Assert(conn.Closed, Equals, true)
	case <-time.After(1 * time.Second):
		c.Error("Connection never disconnected.")
	}
}

func (s *CSuite) TestConnectionErrOrOKReturnsErrorOnDisconnect(c *C) {
	conn := &fakeConn{
		ReadBuffer:  bytes.NewBuffer([]byte{}),
		WriteBuffer: bytes.NewBuffer([]byte{}),
		WriteChan:   make(chan []byte),
	}

	// fill in a fake connection
	s.Connection.conn = conn

	errOrOK := make(chan error)

	go func() {
		errOrOK <- s.Connection.ErrOrOK()
	}()

	go s.Connection.receivePackets()

	select {
	case <-s.Connection.Disconnected:
	case <-time.After(1 * time.Second):
		c.Error("Connection never disconnected.")
	}

	select {
	case err := <-errOrOK:
		c.Assert(err, ErrorMatches, "disconnected")
	case <-time.After(1 * time.Second):
		c.Error("Never received result from ErrOrOK.")
	}
}

func (s *CSuite) TestConnectionOnMessageCallback(c *C) {
	conn := &fakeConn{
		ReadBuffer:  bytes.NewBuffer([]byte("MSG foo 1 5\r\nhello\r\n")),
		WriteBuffer: bytes.NewBuffer([]byte{}),
		WriteChan:   make(chan []byte),
		Closed:      false,
	}

	// fill in a fake connection
	s.Connection.conn = conn

	messages := make(chan *MsgPacket)

	s.Connection.OnMessage(func(msg *MsgPacket) {
		messages <- msg
	})

	go s.Connection.receivePackets()

	select {
	case msg := <-messages:
		c.Assert(msg.SubID, Equals, int64(1))
		c.Assert(string(msg.Payload), Equals, "hello")
	case <-time.After(1 * time.Second):
		c.Error("Did not receive message.")
	}
}

func (s *CSuite) TestConnectionClusterReconnectsAnother(c *C) {
	lock := sync.Mutex{}
	waitGroup := sync.WaitGroup{}
	hellos := 0
	thanks := 0
	goodbyes := 0
	errs := 0

	nodes := [](*FakeConnectionProvider){
		&FakeConnectionProvider{
			ReadBuffer:   "+OK\r\nMSG foo 1 5\r\nhello\r\n",
			WriteBuffer:  []byte{},
			ReturnsError: false,
		},
		&FakeConnectionProvider{
			ReadBuffer:   "+OK\r\nMSG foo 1 5\r\nthank\r\n",
			WriteBuffer:  []byte{},
			ReturnsError: false,
		},
		&FakeConnectionProvider{
			ReadBuffer:   "+OK\r\nMSG foo 1 7\r\ngoodbye\r\n",
			WriteBuffer:  []byte{},
			ReturnsError: false,
		},
	}

	for i := 0; i < 4; i++ {
		if i > 0 {
			nodes[i-1].ReturnsError = true
		}

		cluster := &ConnectionCluster{[]ConnectionProvider{nodes[0], nodes[1], nodes[2]}}

		conn, err := cluster.ProvideConnection()

		if err != nil {
			c.Assert(err.Error(), Equals, "error on dialing")
			errs += 1
		} else {
			waitGroup.Add(1)
			conn.OnMessage(func(msg *MsgPacket) {
				lock.Lock()
				defer lock.Unlock()

				if string(msg.Payload) == "hello" {
					hellos++
				}
				if string(msg.Payload) == "thank" {
					thanks++
				}
				if string(msg.Payload) == "goodbye" {
					goodbyes++
				}
				waitGroup.Done()
			})

			conn.ErrOrOK()
		}
	}

	waitGroup.Wait()

	lock.Lock()
	defer lock.Unlock()

	c.Assert(hellos, Equals, 1)
	c.Assert(thanks, Equals, 1)
	c.Assert(goodbyes, Equals, 1)
	c.Assert(errs, Equals, 1)
}
