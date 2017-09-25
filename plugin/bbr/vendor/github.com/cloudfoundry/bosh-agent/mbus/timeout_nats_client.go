package mbus

import (
	"code.cloudfoundry.org/clock"
	"github.com/cloudfoundry/yagnats"
	"time"
)

var _ yagnats.NATSClient = &TimeoutNatsClient{}

type TimeoutNatsClient struct {
	client yagnats.NATSClient
	clock  clock.Clock
}

func NewTimeoutNatsClient(client yagnats.NATSClient, clock clock.Clock) *TimeoutNatsClient {
	return &TimeoutNatsClient{
		client: client,
		clock:  clock,
	}
}

func (c *TimeoutNatsClient) Ping() bool {
	complete := make(chan bool)
	go func() {
		complete <- c.client.Ping()
	}()

	timeout := c.clock.NewTimer(5 * time.Minute)

	select {
	case success := <-complete:
		timeout.Stop()
		return success
	case <-timeout.C():
		panic("Connect call to NATSClient took too long, exiting so connections are reset")
	}
}

func (c *TimeoutNatsClient) Connect(connectionProvider yagnats.ConnectionProvider) error {
	complete := make(chan error)
	go func() {
		complete <- c.client.Connect(connectionProvider)
	}()

	timeout := c.clock.NewTimer(5 * time.Minute)

	select {
	case err := <-complete:
		timeout.Stop()
		return err
	case <-timeout.C():
		panic("Connect call to NATSClient took too long, exiting so connections are reset")
	}
}

func (c *TimeoutNatsClient) Disconnect() {
	complete := make(chan bool)
	go func() {
		c.client.Disconnect()
		complete <- true
	}()

	timeout := c.clock.NewTimer(5 * time.Minute)

	select {
	case <-complete:
		timeout.Stop()
		return
	case <-timeout.C():
		panic("Disconnect call to NATSClient took too long, exiting so connections are reset")
	}
}

func (c *TimeoutNatsClient) Publish(subject string, payload []byte) error {
	complete := make(chan error)
	go func() {
		complete <- c.client.Publish(subject, payload)
	}()

	timeout := c.clock.NewTimer(5 * time.Minute)

	select {
	case err := <-complete:
		timeout.Stop()
		return err
	case <-timeout.C():
		panic("Publish call to NATSClient took too long, exiting so connections are reset")
	}
}

func (c *TimeoutNatsClient) PublishWithReplyTo(subject, reply string, payload []byte) error {
	panic("not implemented")
}

func (c *TimeoutNatsClient) Subscribe(subject string, callback yagnats.Callback) (int64, error) {
	type result struct {
		subscriberID int64
		err          error
	}

	complete := make(chan result)
	go func() {
		id, err := c.client.Subscribe(subject, callback)
		complete <- result{
			subscriberID: id,
			err:          err,
		}
	}()

	timeout := c.clock.NewTimer(5 * time.Minute)

	select {
	case result := <-complete:
		timeout.Stop()
		return result.subscriberID, result.err
	case <-timeout.C():
		panic("Subscribe call to NATSClient took too long, exiting so connections are reset")
	}
}

func (c *TimeoutNatsClient) SubscribeWithQueue(subject, queue string, callback yagnats.Callback) (int64, error) {
	type result struct {
		subscriberID int64
		err          error
	}

	complete := make(chan result)
	go func() {
		id, err := c.client.SubscribeWithQueue(subject, queue, callback)
		complete <- result{
			subscriberID: id,
			err:          err,
		}
	}()

	timeout := c.clock.NewTimer(5 * time.Minute)

	select {
	case result := <-complete:
		timeout.Stop()
		return result.subscriberID, result.err
	case <-timeout.C():
		panic("SubscribeWithQueue call to NATSClient took too long, exiting so connections are reset")
	}
}

func (c *TimeoutNatsClient) Unsubscribe(subscription int64) error {
	complete := make(chan error)
	go func() {
		complete <- c.client.Unsubscribe(subscription)
	}()

	timeout := c.clock.NewTimer(5 * time.Minute)

	select {
	case err := <-complete:
		timeout.Stop()
		return err
	case <-timeout.C():
		panic("Unsubscribe call to NATSClient took too long, exiting so connections are reset")
	}
}

func (c *TimeoutNatsClient) UnsubscribeAll(subject string) {
	complete := make(chan bool)
	go func() {
		c.client.UnsubscribeAll(subject)
		complete <- true
	}()

	timeout := c.clock.NewTimer(5 * time.Minute)

	select {
	case <-complete:
		timeout.Stop()
		return
	case <-timeout.C():
		panic("UnsubscribeAll call to NATSClient took too long, exiting so connections are reset")
	}
}

func (c *TimeoutNatsClient) BeforeConnectCallback(callback func()) {
	c.client.BeforeConnectCallback(callback)
}
