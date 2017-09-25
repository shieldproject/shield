package mbus

import (
	"net/url"

	"github.com/cloudfoundry/yagnats"

	"code.cloudfoundry.org/clock"
	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type HandlerProvider struct {
	settingsService boshsettings.Service
	logger          boshlog.Logger
	auditLogger     boshplatform.AuditLogger
	handler         boshhandler.Handler
}

func NewHandlerProvider(
	settingsService boshsettings.Service,
	logger boshlog.Logger,
	auditLogger boshplatform.AuditLogger,
) (p HandlerProvider) {
	p.settingsService = settingsService
	p.logger = logger
	p.auditLogger = auditLogger
	return
}

func (p HandlerProvider) Get(
	platform boshplatform.Platform,
	dirProvider boshdir.Provider,
) (handler boshhandler.Handler, err error) {
	if p.handler != nil {
		handler = p.handler
		return
	}

	mbusURL, err := url.Parse(p.settingsService.GetSettings().Mbus)
	if err != nil {
		err = bosherr.WrapError(err, "Parsing handler URL")
		return
	}

	switch mbusURL.Scheme {
	case "nats":
		natsClient := NewTimeoutNatsClient(yagnats.NewClient(), clock.NewClock())
		handler = NewNatsHandler(p.settingsService, natsClient, p.logger, platform)
	case "https":
		mbusKeyPair := p.settingsService.GetSettings().Env.Bosh.Mbus.Cert
		handler = NewHTTPSHandler(mbusURL, mbusKeyPair, p.logger, platform.GetFs(), dirProvider, p.auditLogger)
	default:
		err = bosherr.Errorf("Message Bus Handler with scheme %s could not be found", mbusURL.Scheme)
	}

	p.handler = handler

	return
}
