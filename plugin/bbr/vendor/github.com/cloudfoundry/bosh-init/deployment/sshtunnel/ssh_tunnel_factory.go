package sshtunnel

import (
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/pivotal-golang/clock"
)

type Options struct {
	Host       string
	Port       int
	User       string
	PrivateKey string
	Password   string

	LocalForwardPort  int
	RemoteForwardPort int
}

func (o Options) IsEmpty() bool {
	return o == Options{}
}

type Factory interface {
	NewSSHTunnel(Options) SSHTunnel
}

type factory struct {
	logger boshlog.Logger
}

func NewFactory(logger boshlog.Logger) Factory {
	return &factory{
		logger: logger,
	}
}

func (s *factory) NewSSHTunnel(options Options) SSHTunnel {
	timeService := clock.NewClock()
	return &sshTunnel{
		connectionRefusedTimeout: 5 * time.Minute,
		authFailureTimeout:       2 * time.Minute,
		startDialDelay:           500 * time.Millisecond,
		timeService:              timeService,
		options:                  options,
		logger:                   s.logger,
		logTag:                   "sshTunnel",
	}
}
