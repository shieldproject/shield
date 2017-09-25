package yagnats

import (
	"sync"
	"time"

	"github.com/nats-io/nats"

	. "gopkg.in/check.v1"
)

func (s *YSuite) TestApceraConnectWithOptions(c *C) {
	opts := DefaultOptions()
	opts.PingInterval = time.Duration(30) * time.Second
	urls := []string{"nats://nats:nats@127.0.0.1:4223"}

	client := Must(ConnectWithOptions(urls, opts))

	c.Assert(client.Options().PingInterval, Equals, opts.PingInterval)
}

func (s *YSuite) TestDefaultOptions(c *C) {
	opts := DefaultOptions()
	c.Assert(opts.ReconnectWait, Equals, 500*time.Millisecond)
	c.Assert(opts.MaxReconnect, Equals, -1)
}

func (s *YSuite) TestApceraCloseOnNewClient(c *C) {
	client := Must(Connect([]string{"nats://nats:nats@127.0.0.1:4223"}))
	client.Close()
}

func (s *YSuite) TestApceraConnectWithInvalidAddress(c *C) {
	_, err := Connect([]string{""})

	c.Assert(err, Not(Equals), nil)
	c.Assert(err.Error(), Equals, nats.ErrNoServers.Error())
}

func (s *YSuite) TestApceraClientConnectWithInvalidAuth(c *C) {
	_, err := Connect([]string{"nats://cats:bats@127.0.0.1:4223"})

	c.Assert(err, Not(Equals), nil)
}

func (s *YSuite) TestApceraClientPing(c *C) {
	c.Assert(s.NatsConn.Ping(), Equals, true)
}

func (s *YSuite) TestApceraClientPingWhenNotConnected(c *C) {
	disconnectedClient := Must(Connect([]string{"nats://nats:nats@127.0.0.1:4223"}))
	disconnectedClient.Close()
	c.Assert(disconnectedClient.Ping(), Equals, false)
}

func (s *YSuite) TestApceraClientPingWhenConnectionClosed(c *C) {
	s.NatsConn.Close()
	c.Assert(s.NatsConn.Ping(), Equals, false)
}

func (s *YSuite) TestApceraClientReconnectCB(c *C) {
	sem := make(chan bool)
	reconnectedClient := Must(Connect([]string{"nats://nats:nats@127.0.0.1:4223"}))
	reconnectedClient.AddReconnectedCB(func(_ *nats.Conn) {
		sem <- true
	})

	stopCmd(s.NatsCmd)
	s.NatsCmd = startNats(4223)
	waitUntilNatsUp(4223)

	select {
	case <-sem:
		c.Succeed()
	case <-time.After(1 * time.Second):
		c.Fatal("expected reconnect to be called, it was not")
	}
}

func (s *YSuite) TestApceraClientClosdCB(c *C) {
	closeChannel := make(chan []byte)

	closedClient := Must(Connect([]string{"nats://nats:nats@127.0.0.1:4223"}))
	closedClient.AddClosedCB(func(_ *nats.Conn) {
		closeChannel <- []byte("closed")
	})
	closedClient.Close()

	waitReceive(c, "closed", closeChannel, 500)
}

func (s *YSuite) TestApceraClientDisconnectedCB(c *C) {
	var wg sync.WaitGroup
	wg.Add(1)
	disconnectCalled := false

	disconnectedClient := Must(Connect([]string{"nats://nats:nats@127.0.0.1:4223"}))
	disconnectedClient.AddDisconnectedCB(func(_ *nats.Conn) {
		defer wg.Done()
		disconnectCalled = true
	})
	disconnectedClient.Close()

	wg.Wait()
	c.Assert(disconnectCalled, Equals, true)
}

func (s *YSuite) TestApceraClientSubscribe(c *C) {
	sub, _ := s.NatsConn.Subscribe("some.subject", func(msg *nats.Msg) {})
	c.Assert(sub.Subject, Equals, "some.subject")
}

func (s *YSuite) TestApceraClientUnsubscribe(c *C) {
	payload1 := make(chan []byte)
	payload2 := make(chan []byte)

	sid1, _ := s.NatsConn.Subscribe("some.subject", func(msg *nats.Msg) {
		payload1 <- msg.Data
	})

	s.NatsConn.Subscribe("some.subject", func(msg *nats.Msg) {
		payload2 <- msg.Data
	})

	s.NatsConn.Publish("some.subject", []byte("hello!"))

	waitReceive(c, "hello!", payload1, 500)
	waitReceive(c, "hello!", payload2, 500)

	s.NatsConn.Unsubscribe(sid1)

	s.NatsConn.Publish("some.subject", []byte("hello!"))

	select {
	case <-payload1:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}

	waitReceive(c, "hello!", payload2, 500)
}

func (s *YSuite) TestApceraClientSubscribeAndUnsubscribe(c *C) {
	payload := make(chan []byte)

	sid1, _ := s.NatsConn.Subscribe("some.subject", func(msg *nats.Msg) {
		payload <- msg.Data
	})

	s.NatsConn.Publish("some.subject", []byte("hello!"))

	waitReceive(c, "hello!", payload, 500)

	s.NatsConn.Unsubscribe(sid1)

	s.NatsConn.Subscribe("some.subject", func(msg *nats.Msg) {
		payload <- msg.Data
	})

	s.NatsConn.Publish("some.subject", []byte("hello!"))

	waitReceive(c, "hello!", payload, 500)

	select {
	case <-payload:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *YSuite) TestApceraClientPubSub(c *C) {
	payload := make(chan []byte)

	s.NatsConn.Subscribe("some.subject", func(msg *nats.Msg) {
		payload <- msg.Data
	})

	s.NatsConn.Publish("some.subject", []byte("hello!"))

	waitReceive(c, "hello!", payload, 500)
}

func (s *YSuite) TestApceraClientPubSubWithQueue(c *C) {
	payload := make(chan []byte)

	s.NatsConn.QueueSubscribe("some.subject", "some-queue", func(msg *nats.Msg) {
		payload <- msg.Data
	})

	s.NatsConn.QueueSubscribe("some.subject", "some-queue", func(msg *nats.Msg) {
		payload <- msg.Data
	})

	s.NatsConn.Publish("some.subject", []byte("hello!"))

	waitReceive(c, "hello!", payload, 500)

	select {
	case <-payload:
		c.Error("Should not have received message.")
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *YSuite) TestApceraClientPublishWithReply(c *C) {
	payload := make(chan []byte)

	s.NatsConn.Subscribe("some.request", func(msg *nats.Msg) {
		s.NatsConn.Publish(msg.Reply, []byte("response!"))
	})

	s.NatsConn.Subscribe("some.reply", func(msg *nats.Msg) {
		payload <- msg.Data
	})

	s.NatsConn.PublishRequest("some.request", "some.reply", []byte("hello!"))

	waitReceive(c, "response!", payload, 500)
}

func Must(conn NATSConn, e error) NATSConn {
	if e != nil {
		panic("Expected no error, got " + e.Error())
	}
	return conn
}
