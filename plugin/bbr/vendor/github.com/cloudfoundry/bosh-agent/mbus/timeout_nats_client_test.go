package mbus_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/yagnats"

	"code.cloudfoundry.org/clock/fakeclock"
	. "github.com/cloudfoundry/bosh-agent/mbus"
	. "github.com/cloudfoundry/bosh-agent/mbus/fakes"
	"github.com/cloudfoundry/bosh-utils/errors"
	"reflect"
	"time"
)

func init() {
	Describe("TimeoutNatsClient", func() {
		var (
			client   yagnats.NATSClient
			delegate *FakeNATSClient
			clock    *fakeclock.FakeClock
		)

		BeforeEach(func() {
			delegate = &FakeNATSClient{}
			clock = fakeclock.NewFakeClock(time.Now())
			client = NewTimeoutNatsClient(delegate, clock)
		})

		It("delegates Ping", func() {
			delegate.PingReturns(true)
			Expect(client.Ping()).To(Equal(true))
			Expect(delegate.PingCallCount()).To(Equal(1))
		})

		It("delegates Connect", func() {
			err := errors.Error("OMG")
			delegate.ConnectReturns(err)

			provider := &FakeConnectionProvider{}
			Expect(client.Connect(provider)).To(Equal(err))
			Expect(delegate.ConnectArgsForCall(0)).To(Equal(provider))
		})

		It("delegates Disconnect", func() {
			client.Disconnect()
			Expect(delegate.DisconnectCallCount()).To(Equal(1))
		})

		It("delegates Publish", func() {
			err := errors.Error("OMG")
			payload := []byte{0, 0, 0}
			delegate.PublishReturns(err)
			Expect(client.Publish("subject", payload)).To(Equal(err))
			delegatedSubject, delegatedPayload := delegate.PublishArgsForCall(0)
			Expect(delegatedSubject).To(Equal("subject"))
			Expect(delegatedPayload).To(Equal(delegatedPayload))
		})

		It("PublishWithReplyTo panics because it's unused and not implemented", func() {
			payload := []byte{0, 0, 0}
			Expect(func() { client.PublishWithReplyTo("subject", "reply", payload) }).To(Panic())
		})

		It("delegates Subscribe", func() {
			err := errors.Error("OMG")
			delegate.SubscribeReturns(int64(42), err)
			var callback yagnats.Callback
			callback = func(message *yagnats.Message) {}
			id, delegateErr := client.Subscribe("subject", callback)
			Expect(id).To(Equal(int64(42)))
			Expect(delegateErr).To(Equal(err))
			subject, delegateCallback := delegate.SubscribeArgsForCall(0)
			Expect(subject).To(Equal(subject))
			Expect(reflect.ValueOf(delegateCallback)).To(Equal(reflect.ValueOf(callback)))
		})

		It("delegates SubscribeWithQueue", func() {
			err := errors.Error("OMG")
			delegate.SubscribeWithQueueReturns(int64(42), err)
			var callback yagnats.Callback
			callback = func(message *yagnats.Message) {}
			id, delegateErr := client.SubscribeWithQueue("subject", "queue", callback)
			Expect(id).To(Equal(int64(42)))
			Expect(delegateErr).To(Equal(err))
			subject, queue, delegateCallback := delegate.SubscribeWithQueueArgsForCall(0)
			Expect(subject).To(Equal(subject))
			Expect(queue).To(Equal(queue))
			Expect(reflect.ValueOf(delegateCallback)).To(Equal(reflect.ValueOf(callback)))
		})

		It("delegates Unsubscribe", func() {
			err := errors.Error("OMG")
			delegate.UnsubscribeReturns(err)
			Expect(client.Unsubscribe(42)).To(Equal(err))
			Expect(delegate.UnsubscribeArgsForCall(0)).To(Equal(int64(42)))
		})

		It("delegates UnsubscribeAll", func() {
			client.UnsubscribeAll("subject")
			Expect(delegate.UnsubscribeAllArgsForCall(0)).To(Equal("subject"))
		})

		It("delegates BeforeConnectCallback", func() {
			callback := func() {}
			client.BeforeConnectCallback(callback)
			thing := delegate.BeforeConnectCallbackArgsForCall(0)
			Expect(reflect.ValueOf(thing)).To(Equal(reflect.ValueOf(callback)))
		})

		Describe("slow RPC calls", func() {
			Context("Ping", func() {
				Context("when the call takes more than 5min", func() {
					It("panics", func() {
						delegate.PingStub = func() bool {
							clock.Increment(301 * time.Second)
							return false
						}

						Expect(func() { client.Ping() }).To(Panic())
					})
				})

				Context("when the call takes less than 5min", func() {
					It("does not panic and releases any lagging timers", func() {
						delegate.PingStub = func() bool {
							clock.Increment(299 * time.Second)
							return false
						}

						Expect(func() { client.Ping() }).ToNot(Panic())
						Expect(clock.WatcherCount()).To(Equal(0))
					})
				})
			})

			Context("Connect", func() {
				Context("when the call takes more than 5min", func() {
					It("panics", func() {
						delegate.ConnectStub = func(connectionProvider yagnats.ConnectionProvider) error {
							clock.Increment(301 * time.Second)
							return nil
						}

						Expect(func() {
							client.Connect(&FakeConnectionProvider{})
						}).To(Panic())
					})
				})

				Context("when the call takes less than 5min", func() {
					It("does not panic and releases any lagging timers", func() {
						delegate.ConnectStub = func(connectionProvider yagnats.ConnectionProvider) error {
							clock.Increment(299 * time.Second)
							return nil
						}

						Expect(func() { client.Connect(&FakeConnectionProvider{}) }).ToNot(Panic())
						Expect(clock.WatcherCount()).To(Equal(0))
					})
				})
			})

			Context("Disconnect", func() {
				Context("when the call takes more than 5min", func() {
					It("panics", func() {
						delegate.DisconnectStub = func() {
							clock.Increment(301 * time.Second)
						}

						Expect(func() {
							client.Disconnect()
						}).To(Panic())
					})
				})

				Context("when the call takes less than 5min", func() {
					It("does not panic and releases any lagging timers", func() {
						delegate.DisconnectStub = func() {
							clock.Increment(299 * time.Second)
						}

						Expect(func() { client.Disconnect() }).ToNot(Panic())
						Expect(clock.WatcherCount()).To(Equal(0))
					})
				})
			})

			Context("Publish", func() {
				Context("when the call takes more than 5min", func() {
					It("panics", func() {
						delegate.PublishStub = func(string, []byte) error {
							clock.Increment(301 * time.Second)
							return nil
						}

						Expect(func() { client.Publish("subject", []byte{0}) }).To(Panic())
					})
				})

				Context("when the call takes less than 5min", func() {
					It("does not panic and releases any lagging timers", func() {
						delegate.PublishStub = func(string, []byte) error {
							clock.Increment(299 * time.Second)
							return nil
						}

						Expect(func() { client.Publish("subject", []byte{0}) }).ToNot(Panic())
						Expect(clock.WatcherCount()).To(Equal(0))
					})
				})
			})

			Context("Subscribe", func() {
				Context("when the call takes more than 5min", func() {
					It("panics", func() {
						delegate.SubscribeStub = func(subject string, callback yagnats.Callback) (int64, error) {
							clock.Increment(301 * time.Second)
							return 42, nil
						}

						Expect(func() { client.Subscribe("subject", func(message *yagnats.Message) {}) }).To(Panic())
					})
				})

				Context("when the call takes less than 5min", func() {
					It("does not panic and releases any lagging timers", func() {
						delegate.SubscribeStub = func(string, yagnats.Callback) (int64, error) {
							clock.Increment(299 * time.Second)
							return 42, nil
						}

						Expect(func() { client.Subscribe("subject", func(*yagnats.Message) {}) }).ToNot(Panic())
						Expect(clock.WatcherCount()).To(Equal(0))
					})
				})
			})

			Context("SubscribeWithQueue", func() {
				Context("when the call takes more than 5min", func() {
					It("panics", func() {
						delegate.SubscribeWithQueueStub = func(string, string, yagnats.Callback) (int64, error) {
							clock.Increment(301 * time.Second)
							return 42, nil
						}

						Expect(func() {
							client.SubscribeWithQueue("subject", "queue", func(*yagnats.Message) {})
						}).To(Panic())
					})
				})

				Context("when the call takes less than 5min", func() {
					It("does not panic and releases any lagging timers", func() {
						delegate.SubscribeWithQueueStub = func(string, string, yagnats.Callback) (int64, error) {
							clock.Increment(299 * time.Second)
							return 42, nil
						}

						Expect(func() {
							client.SubscribeWithQueue("subject", "queue", func(*yagnats.Message) {})
						}).ToNot(Panic())
						Expect(clock.WatcherCount()).To(Equal(0))
					})
				})
			})

			Context("Unsubscribe", func() {
				Context("when the call takes more than 5min", func() {
					It("panics", func() {
						delegate.UnsubscribeStub = func(int64) error {
							clock.Increment(301 * time.Second)
							return nil
						}

						Expect(func() { client.Unsubscribe(42) }).To(Panic())
					})
				})

				Context("when the call takes less than 5min", func() {
					It("does not panic and releases any lagging timers", func() {
						delegate.UnsubscribeStub = func(int64) error {
							clock.Increment(299 * time.Second)
							return nil
						}

						Expect(func() { client.Unsubscribe(42) }).ToNot(Panic())
						Expect(clock.WatcherCount()).To(Equal(0))
					})
				})
			})

			Context("UnsubscribeAll", func() {
				Context("when the call takes more than 5min", func() {
					It("panics", func() {
						delegate.UnsubscribeAllStub = func(string) {
							clock.Increment(301 * time.Second)
						}

						Expect(func() { client.UnsubscribeAll("banana") }).To(Panic())
					})
				})

				Context("when the call takes less than 5min", func() {
					It("does not panic and releases any lagging timers", func() {
						delegate.UnsubscribeAllStub = func(string) {
							clock.Increment(299 * time.Second)
						}

						Expect(func() { client.UnsubscribeAll("banana") }).ToNot(Panic())
						Expect(clock.WatcherCount()).To(Equal(0))
					})
				})
			})
		})
	})
}
